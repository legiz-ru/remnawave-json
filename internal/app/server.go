package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"remnawave-json/internal/config"
	"remnawave-json/internal/transport/rest"
	"strings"
	"time"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
)

type SingboxVersion struct {
	Major int
	Minor int
	Patch int
}

var server *http.Server

func Start() {
	r := mux.NewRouter()

	r.Use(httpsAndProxyMiddleware)

	r.HandleFunc("/{shortUuid}", userAgentRouter()).Methods(http.MethodGet)
	r.HandleFunc("/{shortUuid}/v2ray-json", v2rayJson()).Methods(http.MethodGet)

	// Добавить маршруты для всех версий sing-box:
	r.HandleFunc("/{shortUuid}/singbox-json-1_7", singboxJson1_7()).Methods(http.MethodGet)
	r.HandleFunc("/{shortUuid}/singbox-json-1_9", singboxJson1_9()).Methods(http.MethodGet)
	r.HandleFunc("/{shortUuid}/singbox-json-1_10", singboxJson1_10()).Methods(http.MethodGet)
	r.HandleFunc("/{shortUuid}/singbox-json-1_11", singboxJson1_11()).Methods(http.MethodGet)
	r.HandleFunc("/{shortUuid}/singbox-json-1_12", singboxJson1_12()).Methods(http.MethodGet)

	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("/app/templates/assets"))))

	server = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", config.GetAppHost(), config.GetAppPort()),
		Handler: r,
	}

	slog.Info("Starting server on http://" + server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Error while starting server")
		panic(err)
	}
}

func v2rayJson() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest.V2rayJson(w, r)
		return
	}
}

func singboxJson1_7() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest.SingboxJson1_7(w, r)
		return
	}
}

func singboxJson1_9() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest.SingboxJson1_9(w, r)
		return
	}
}

func singboxJson1_10() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest.SingboxJson1_10(w, r)
		return
	}
}

func singboxJson1_11() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest.SingboxJson1_11(w, r)
		return
	}
}

func singboxJson1_12() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest.SingboxJson1_12(w, r)
		return
	}
}

func Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Error during server shutdown", "error", err)
		if err = server.Close(); err != nil {
			slog.Error("Error during server shutdown", "error", err)
			panic(err)
		}
	}
}

func httpsAndProxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.GetAppHost() == "localhost" {
			next.ServeHTTP(w, r)
			return
		}
		xForwardedFor := r.Header.Get("X-Forwarded-For")
		xForwardedProto := r.Header.Get("X-Forwarded-Proto")

		if xForwardedFor == "" || xForwardedProto != "https" {
			slog.Error("Reverse proxy and HTTPS are required.")
			http.Error(w, "Reverse proxy and HTTPS are required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isSingBoxClient(userAgent string) bool {
	return strings.HasPrefix(userAgent, "SFA/") ||
		strings.HasPrefix(userAgent, "SFI/") ||
		strings.HasPrefix(userAgent, "SFM/") ||
		strings.HasPrefix(userAgent, "SFT/") ||
		strings.Contains(userAgent, "Dart/3.4")
}

func parseSingboxVersionFromUA(userAgent string) SingboxVersion {
	// Извлекаем версию sing-box из строк вида "sing-box 1.11.4"
	re := regexp.MustCompile(`sing-box (\d+)\.(\d+)\.?(\d*)`)
	matches := re.FindStringSubmatch(userAgent)

	if len(matches) >= 3 {
		major, _ := strconv.Atoi(matches[1])
		minor, _ := strconv.Atoi(matches[2])
		patch := 0
		if len(matches) > 3 && matches[3] != "" {
			patch, _ = strconv.Atoi(matches[3])
		}
		return SingboxVersion{Major: major, Minor: minor, Patch: patch}
	}

	// Fallback на последнюю версию если парсинг не удался
	return SingboxVersion{Major: 1, Minor: 12, Patch: 0}
}

func userAgentRouter() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		
		if isBrowser(userAgent) {
			rest.WebPage(w, r)
			return
		}
		
		if strings.Contains(userAgent, "Streisand") {
			rest.V2rayJson(w, r)
			return
		}

		// Обработка sing-box клиентов с автоматическим определением версии
		if isSingBoxClient(userAgent) {
			version := parseSingboxVersionFromUA(userAgent)
			
			switch {
			case strings.Contains(userAgent, "Dart/3.4"):
				// Dart/3.4 всегда использует конфигурацию версии 1.9
				rest.SingboxJson1_9(w, r)
				return
			case version.Major == 1 && version.Minor < 8:
				rest.SingboxJson1_7(w, r)
				return
			case version.Major == 1 && version.Minor < 10:
				rest.SingboxJson1_9(w, r)
				return
			case version.Major == 1 && version.Minor < 11:
				rest.SingboxJson1_10(w, r)
				return
			case version.Major == 1 && version.Minor < 12:
				rest.SingboxJson1_11(w, r)
				return
			default:
				// Последняя версия по умолчанию
				rest.SingboxJson1_12(w, r)
				return
			}
		}

		if strings.Contains(userAgent, "Happ") && config.IsHappJsonEnabled() {
			rest.HappJson(w, r)
			return
		}

		rest.Direct(w, r)
	}
}

var browserKeywords = [...]string{"Mozilla", "Chrome", "Safari", "Firefox", "Opera", "Edge", "TelegramBot"}

func isBrowser(userAgent string) bool {
	for _, keyword := range browserKeywords {
		if strings.Contains(userAgent, keyword) {
			return true
		}
	}
	return false
}
