package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"
)

type SingboxConfigData struct {
	V1_7         bool     `json:"v1_7"`
	V1_9         bool     `json:"v1_9"`
	V1_10        bool     `json:"v1_10"`
	V1_11        bool     `json:"v1_11"`
	V1_12        bool     `json:"v1_12"`
	IsDart       bool     `json:"is_dart"`
	UserUUID     string   `json:"user_uuid"`
	Country      string   `json:"country"`
	Outbounds    []Outbound `json:"outbounds"`
	OutboundsTags []string   `json:"outboundstags"`
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
	const templatePath = "/app/templates/singbox/singbox_template.json"
	content, err := ioutil.ReadFile(templatePath)
	if err != nil {
		panic(fmt.Sprintf("Failed to read singbox template from %s: %v", templatePath, err))
	}
	singboxTemplate, err = template.New("singbox").Parse(string(content))
	if err != nil {
		panic("Failed to parse singbox template: " + err.Error())
	}
}

func fetchOutbounds(shortUuid string) ([]Outbound, []string, error) {
	apiURL := fmt.Sprintf("http://localhost/api/sub/%s/singbox", shortUuid)
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch outbounds: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	var filteredOutbounds []Outbound
	var outboundTags []string
	for _, outbound := range apiResp.Outbounds {
		outboundType, ok := outbound["type"].(string)
		if !ok {
			continue
		}
		tag, tagOk := outbound["tag"].(string)
		if !tagOk {
			continue
		}
		if outboundType == "vless" || outboundType == "trojan" {
			outboundTags = append(outboundTags, tag)
		}
		config := make(map[string]interface{})
		for k, v := range outbound {
			if k != "type" && k != "tag" {
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
	return filteredOutbounds, outboundTags, nil
}

func GetSingboxConfigWithOutbounds(baseData SingboxConfigData, shortUuid string) (string, error) {
	outbounds, outboundsTags, err := fetchOutbounds(shortUuid)
	if err != nil {
		return "", err
	}
	baseData.Outbounds = outbounds
	baseData.OutboundsTags = outboundsTags
	return GetSingboxConfigJSON(baseData)
}

func GetSingboxConfigJSON(data SingboxConfigData) (string, error) {
	var buf bytes.Buffer
	if err := singboxTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
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