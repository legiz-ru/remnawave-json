package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
)

type SingboxConfigData struct {
	// Версионные флаги - аналогично .j2 шаблону
	V1_7  bool `json:"v1_7"`
	V1_9  bool `json:"v1_9"`
	V1_10 bool `json:"v1_10"`
	V1_11 bool `json:"v1_11"`
	V1_12 bool `json:"v1_12"`
	
	// Специальные флаги
	IsDart bool `json:"is_dart"`
	
	// Конфигурационные данные
	UserUUID  string     `json:"user_uuid"`
	Country   string     `json:"country"`
	Outbounds []Outbound `json:"outbounds"`
}

type Outbound struct {
	Type   string                 `json:"type"`
	Tag    string                 `json:"tag"`
	Config map[string]interface{} `json:",inline"`
}

type APIResponse struct {
	Outbounds []map[string]interface{} `json:"outbounds"`
}

var singboxTemplate *template.Template

func init() {
	// Поиск файла шаблона в различных возможных местах
	possiblePaths := []string{
		"templates/singbox_template.json",                    // Относительно рабочей директории
		"/app/templates/singbox_template.json",               // Абсолютный путь в контейнере
		"./templates/singbox_template.json",                  // Текущая директория
		"../templates/singbox_template.json",                 // Уровень выше
		"../../templates/singbox_template.json",              // Два уровня выше
	}

	var templateContent []byte
	var err error
	var foundPath string

	for _, path := range possiblePaths {
		templateContent, err = ioutil.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if err != nil {
		// Если файл не найден, создаем встроенный шаблон
		templateContent = []byte(getEmbeddedTemplate())
		foundPath = "embedded template"
	}

	fmt.Printf("Loading singbox template from: %s\n", foundPath)

	singboxTemplate, err = template.New("singbox").Parse(string(templateContent))
	if err != nil {
		panic("Failed to parse singbox template: " + err.Error())
	}
}

// getEmbeddedTemplate возвращает встроенный шаблон как fallback
func getEmbeddedTemplate() string {
	return `{
    "outbounds": [{{range $index, $outbound := .Outbounds}}{{if $index}},{{end}}
        {
            "type": "{{$outbound.Type}}",
            "tag": "{{$outbound.Tag}}"{{range $key, $value := $outbound.Config}},
            "{{$key}}": {{$value}}{{end}}
        }{{end}},
        {
            "tag": "direct",
            "type": "direct"
        }{{if .V1_10}},
        {
            "tag": "block",
            "type": "block"
        },
        {
            "tag": "dns-out",
            "type": "dns"
        }{{end}},
        {
            "tag": "bypass",
            "type": "direct"
        }
    ],
    "route": {
        "auto_detect_interface": true,
        "override_android_vpn": true,
        "final": "{{if .Outbounds}}{{(index .Outbounds 0).Tag}}{{else}}direct{{end}}"{{if .V1_7}},
        "geoip": {
            "download_url": "https://github.com/SagerNet/sing-geoip/releases/latest/download/geoip.db",
            "download_detour": "bypass"
        },
        "geosite": {
            "download_url": "https://github.com/SagerNet/sing-geosite/releases/latest/download/geosite.db",
            "download_detour": "bypass"
        }{{else}},
        "rule_set": [{{if eq .Country "ru"}}
            {
                "tag": "geosite-ru",
                "type": "remote",
                "format": "binary",
                "url": "https://github.com/MetaCubeX/meta-rules-dat/raw/sing/geo/geosite/category-ru.srs",
                "download_detour": "bypass"
            },
            {
                "tag": "geoip-ru",
                "type": "remote",
                "format": "binary",
                "url": "https://github.com/MetaCubeX/meta-rules-dat/raw/sing/geo/geoip/ru.srs",
                "download_detour": "bypass"
            }{{end}}
        ]{{end}},
        "rules": [{{if .V1_10}}
            {
                "outbound": "dns-out",
                "port": [53]
            },
            {
                "inbound": ["dns-in"],
                "outbound": "dns-out"
            }{{else}}
            {
                "inbound": ["tun-in", "mixed-in"],
                "port": [53],
                "action": "hijack-dns"
            },
            {
                "inbound": ["dns-in"],
                "action": "hijack-dns"
            }{{end}}{{if not .V1_10}},
            {
                "inbound": ["tun-in", "mixed-in"],
                "action": "sniff",
                "timeout": "1s"
            }{{if not .V1_11}},
            {
                "inbound": ["tun-in", "mixed-in"],
                "action": "resolve",
                "strategy": "prefer_ipv4"
            }{{end}}{{end}}{{if eq .Country "ru"}},
            {
                "domain_suffix": ["ru"],
                "outbound": "bypass"
            }{{if .V1_7}},
            {
                "geoip": ["ru"],
                "outbound": "bypass"
            }{{else}},
            {
                "rule_set": "geoip-ru",
                "outbound": "bypass"
            },
            {
                "rule_set": "geosite-ru",
                "outbound": "bypass"
            }{{end}}{{end}},
            {
                "protocol": "quic",
                "port": [443]{{if .V1_10}},
                "outbound": "block"{{else}},
                "action": "reject"{{end}}
            },
            {
                "ip_cidr": ["224.0.0.0/3", "ff00::/8"],
                "source_ip_cidr": ["224.0.0.0/3", "ff00::/8"]{{if .V1_10}},
                "outbound": "block"{{else}},
                "action": "reject"{{end}}
            }
        ]
    },
    "experimental": {
        "clash_api": {
            "external_controller": "127.0.0.1:9090",
            "external_ui_download_url": "https://github.com/MetaCubeX/Yacd-meta/archive/gh-pages.zip",
            "default": "{{if .Outbounds}}{{(index .Outbounds 0).Tag}}{{else}}direct{{end}}"{{if .V1_7}},
            "cache_file": "cache.db",
            "cache_id": "{{.UserUUID}}",
            "store_mode": true,
            "store_selected": true,
            "store_fakeip": true{{end}}
        }{{if not .V1_7}},
        "cache_file": {
            "enabled": true,
            "path": "cache.db",
            "cache_id": "{{.UserUUID}}",
            "store_fakeip": true
        }{{end}}
    },
    "dns": {
        "servers": [{{if .V1_12}}
            {
                "type": "tcp",
                "server": "1.1.1.1",
                "domain_resolver": "dns-local",
                "tag": "dns-remote",
                "detour": "{{if .Outbounds}}{{(index .Outbounds 0).Tag}}{{else}}direct{{end}}"
            },
            {
                "type": "udp",
                "server": "195.208.4.1",
                "detour": "direct",
                "tag": "dns-local"
            }{{else}}
            {
                "address": "tcp://1.1.1.1",
                "address_resolver": "dns-local",
                "strategy": "prefer_ipv4",
                "tag": "dns-remote",
                "detour": "{{if .Outbounds}}{{(index .Outbounds 0).Tag}}{{else}}direct{{end}}"
            },
            {
                "address": "195.208.4.1",
                "detour": "direct",
                "tag": "dns-local"
            },
            {
                "address": "rcode://success",
                "tag": "dns-block"
            }{{end}}
        ],
        "rules": [
            {
                "domain": [
                    "github.com",
                    "githubusercontent.com",
                    "raw.githubusercontent.com",
                    "1.1.1.1"
                ],
                "server": "dns-local"{{if .V1_12}},
                "strategy": "prefer_ipv4"{{end}}
            }{{if eq .Country "ru"}},
            {
                "domain_suffix": ["ru"],
                "server": "dns-local"
            }{{end}}{{if not .V1_12}},
            {
                "outbound": "direct",
                "server": "dns-local"
            }{{end}}
        ],
        "final": "dns-remote",
        "reverse_mapping": true,
        "independent_cache": true{{if not .V1_12}},
        "strategy": "prefer_ipv4"{{else}},
        "strategy": "prefer_ipv4"{{end}}
    },
    "inbounds": [{{if .V1_11}}
        {
            "listen": "127.0.0.1",
            "listen_port": 6450,
            "override_address": "195.208.4.1",
            "override_port": 53,
            "tag": "dns-in",
            "type": "direct"
        },{{end}}
        {
            "type": "tun",
            "tag": "tun-in",
            "interface_name": "tun0",{{if .V1_9}}
            "inet4_address": "172.19.0.1/30",{{else}}
            "address": ["172.19.0.1/30"],{{end}}
            "mtu": 9000,
            "auto_route": true,
            "strict_route": true,
            "stack": "system",{{if .V1_10}}
            "sniff": true,
            "domain_strategy": "prefer_ipv4",
            "sniff_override_destination": false,{{end}}
            "endpoint_independent_nat": true
        },
        {
            "listen": "127.0.0.1",
            "listen_port": 2334,{{if .V1_10}}
            "domain_strategy": "prefer_ipv4",
            "sniff": true,
            "sniff_override_destination": false,{{end}}
            "tag": "mixed-in",
            "type": "mixed"
        }
    ]
}`
}

// fetchOutbounds получает outbounds из API /api/sub/{shortUuid}/singbox
func fetchOutbounds(shortUuid string) ([]Outbound, error) {
	// Формируем URL для запроса
	apiURL := fmt.Sprintf("http://localhost/api/sub/%s/singbox", shortUuid)
	
	// Выполняем HTTP запрос
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch outbounds: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	
	// Читаем ответ
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Парсим JSON
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	// Фильтруем outbounds по типу (только vless и trojan)
	var filteredOutbounds []Outbound
	for _, outbound := range apiResp.Outbounds {
		outboundType, ok := outbound["type"].(string)
		if !ok {
			continue
		}
		
		if outboundType == "vless" || outboundType == "trojan" {
			tag, tagOk := outbound["tag"].(string)
			if !tagOk {
				continue
			}
			
			// Создаем копию конфигурации без type и tag
			config := make(map[string]interface{})
			for k, v := range outbound {
				if k != "type" && k != "tag" {
					// Преобразуем значение в JSON строку для вставки в шаблон
					jsonValue, err := json.Marshal(v)
					if err != nil {
						continue
					}
					config[k] = string(jsonValue)
				}
			}
			
			filteredOutbounds = append(filteredOutbounds, Outbound{
				Type:   outboundType,
				Tag:    tag,
				Config: config,
			})
		}
	}
	
	return filteredOutbounds, nil
}

// GenerateSingboxConfig генерирует конфигурацию sing-box на основе шаблона
func GenerateSingboxConfig(data SingboxConfigData) (map[string]interface{}, error) {
	var buf bytes.Buffer
	
	// Выполняем рендеринг шаблона
	if err := singboxTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	
	// Парсим результат в map для возврата
	var config map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &config); err != nil {
		return nil, err
	}
	
	return config, nil
}

// GetSingboxConfigJSON возвращает JSON строку конфигурации
func GetSingboxConfigJSON(data SingboxConfigData) (string, error) {
	var buf bytes.Buffer
	
	if err := singboxTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	
	// Форматируем JSON для красивого вывода
	var temp interface{}
	if err := json.Unmarshal(buf.Bytes(), &temp); err != nil {
		return "", err
	}
	
	formatted, err := json.MarshalIndent(temp, "", "    ")
	if err != nil {
		return "", err
	}
	
	return string(formatted), nil
}

// GetSingboxConfigWithOutbounds создает конфигурацию с загруженными outbounds
func GetSingboxConfigWithOutbounds(baseData SingboxConfigData, shortUuid string) (string, error) {
	// Загружаем outbounds из API
	outbounds, err := fetchOutbounds(shortUuid)
	if err != nil {
		// Если не удалось загрузить, возвращаем конфигурацию без прокси outbounds
		fmt.Printf("Warning: failed to fetch outbounds for %s: %v\n", shortUuid, err)
		outbounds = []Outbound{}
	}
	
	// Добавляем outbounds к данным
	baseData.Outbounds = outbounds
	
	// Генерируем конфигурацию
	return GetSingboxConfigJSON(baseData)
}
