package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
	// Загружаем шаблон из файла templates/singbox_template.json
	templatePath := filepath.Join("templates", "singbox_template.json")
	templateContent, err := ioutil.ReadFile(templatePath)
	if err != nil {
		// Пытаемся найти файл относительно корня проекта
		templatePath = filepath.Join("..", "..", "templates", "singbox_template.json")
		templateContent, err = ioutil.ReadFile(templatePath)
		if err != nil {
			panic("Failed to read singbox template from templates/singbox_template.json: " + err.Error())
		}
	}

	singboxTemplate, err = template.New("singbox").Parse(string(templateContent))
	if err != nil {
		panic("Failed to parse singbox template: " + err.Error())
	}
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
	// Для Dart/3.4 всегда используем конфиг sing-box 1.9
	// Никаких дополнительных изменений версионных флагов не требуется
	
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
		outbounds = []Outbound{}
	}
	
	// Добавляем outbounds к данным
	baseData.Outbounds = outbounds
	
	// Генерируем конфигурацию
	return GetSingboxConfigJSON(baseData)
}