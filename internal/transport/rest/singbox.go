package rest

import (
	"net/http"
	"remnawave-json/internal/templates"
	"strings"

	"github.com/gorilla/mux"
)

func SingboxJson1_7(w http.ResponseWriter, r *http.Request) {
	data := templates.SingboxConfigData{
		V1_7:     true,
		V1_9:     false,
		V1_10:    false,
		V1_11:    false,
		V1_12:    false,
		IsDart:   false,
		UserUUID: getShortUUID(r),
		Country:  "ru",
	}
	sendSingboxConfigWithOutbounds(w, data, getShortUUID(r))
}

func SingboxJson1_9(w http.ResponseWriter, r *http.Request) {
	userAgent := r.Header.Get("User-Agent")
	isDart := strings.Contains(userAgent, "Dart/3.4")

	data := templates.SingboxConfigData{
		V1_7:     false,
		V1_9:     true,
		V1_10:    false,
		V1_11:    false,
		V1_12:    false,
		IsDart:   isDart,
		UserUUID: getShortUUID(r),
		Country:  "ru",
	}
	sendSingboxConfigWithOutbounds(w, data, getShortUUID(r))
}

func SingboxJson1_10(w http.ResponseWriter, r *http.Request) {
	data := templates.SingboxConfigData{
		V1_7:     false,
		V1_9:     false,
		V1_10:    true,
		V1_11:    false,
		V1_12:    false,
		IsDart:   false,
		UserUUID: getShortUUID(r),
		Country:  "ru",
	}
	sendSingboxConfigWithOutbounds(w, data, getShortUUID(r))
}

func SingboxJson1_11(w http.ResponseWriter, r *http.Request) {
	data := templates.SingboxConfigData{
		V1_7:     false,
		V1_9:     false,
		V1_10:    false,
		V1_11:    true,
		V1_12:    false,
		IsDart:   false,
		UserUUID: getShortUUID(r),
		Country:  "ru",
	}
	sendSingboxConfigWithOutbounds(w, data, getShortUUID(r))
}

func SingboxJson1_12(w http.ResponseWriter, r *http.Request) {
	data := templates.SingboxConfigData{
		V1_7:     false,
		V1_9:     false,
		V1_10:    false,
		V1_11:    false,
		V1_12:    true,
		IsDart:   false,
		UserUUID: getShortUUID(r),
		Country:  "ru",
	}
	sendSingboxConfigWithOutbounds(w, data, getShortUUID(r))
}

func sendSingboxConfigWithOutbounds(w http.ResponseWriter, data templates.SingboxConfigData, shortUuid string) {
	// Получаем конфигурацию с загруженными outbounds
	configJSON, err := templates.GetSingboxConfigWithOutbounds(data, shortUuid)
	if err != nil {
		http.Error(w, "Error generating config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=config.json")

	// Отправляем готовый JSON
	w.Write([]byte(configJSON))
}

func getShortUUID(r *http.Request) string {
	vars := mux.Vars(r)
	shortUuid := vars["shortUuid"]

	// Здесь можно добавить логику получения полного UUID по shortUuid
	// Пока возвращаем shortUuid как заглушку
	return shortUuid
}