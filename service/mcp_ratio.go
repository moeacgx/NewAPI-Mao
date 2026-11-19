package service

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

// 自定义价格来源不允许借用另一渠道的凭据，绝对 endpoint 也不能覆盖目标站点。
func validateMCPRatioRequest(raw []byte) error {
	var request dto.UpstreamRequest
	if common.Unmarshal(raw, &request) != nil || len(request.Upstreams)+len(request.ChannelIDs) == 0 || len(request.Upstreams)+len(request.ChannelIDs) > 16 {
		return errors.New("provide 1-16 pricing sources")
	}
	for _, upstream := range request.Upstreams {
		// 内置来源使用负数哨兵；其他负数不是有效渠道或预设身份。
		if upstream.ID < 0 && upstream.ID != -100 && upstream.ID != -101 {
			return errors.New("invalid pricing source id")
		}
		address, err := url.Parse(upstream.BaseURL)
		if err != nil || address.Host == "" || address.User != nil || address.RawQuery != "" || address.Fragment != "" || address.Scheme != "http" && address.Scheme != "https" {
			return errors.New("invalid pricing base_url")
		}
		endpoint, err := url.Parse(upstream.Endpoint)
		if err != nil || endpoint.IsAbs() || endpoint.Host != "" || strings.Contains(upstream.Endpoint, "\\") || strings.Contains(upstream.Endpoint, "..") {
			return errors.New("pricing endpoint must be relative to base_url")
		}
		if upstream.Endpoint == "openrouter" {
			if upstream.ID <= 0 || model.DB == nil {
				return errors.New("authenticated pricing source requires an existing channel")
			}
			channel, err := model.GetChannelById(upstream.ID, false)
			if err != nil || strings.TrimRight(channel.GetBaseURL(), "/") != strings.TrimRight(upstream.BaseURL, "/") {
				return errors.New("pricing credential origin does not match the saved channel")
			}
		}
	}
	return nil
}

func projectMCPRatioResponse(tool MCPTool, raw []byte) ([]byte, bool, error) {
	if tool.Name != "newapi_ratio_fetch" && tool.Name != "newapi_ratio_sources" {
		return nil, false, nil
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := common.Unmarshal(raw, &envelope); err != nil {
		return nil, true, err
	}
	if tool.Name == "newapi_ratio_sources" {
		var sources []dto.SyncableChannel
		if err := common.Unmarshal(envelope.Data, &sources); err != nil {
			return nil, true, err
		}
		for i := range sources {
			clean, hidden := mcpScrubConfigValue(sources[i].BaseURL, "")
			if hidden {
				sources[i].BaseURL = ""
			} else {
				sources[i].BaseURL, _ = clean.(string)
			}
		}
		result, err := common.Marshal(map[string]any{"success": true, "data": sources})
		return result, true, err
	}
	var payload struct {
		Differences map[string]map[string]struct {
			Current    json.RawMessage            `json:"current"`
			Upstreams  map[string]json.RawMessage `json:"upstreams"`
			Confidence map[string]bool            `json:"confidence"`
		} `json:"differences"`
		TestResults []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"test_results"`
	}
	if err := common.Unmarshal(envelope.Data, &payload); err != nil {
		return nil, true, err
	}
	allowed := " model_ratio completion_ratio cache_ratio create_cache_ratio image_ratio audio_ratio audio_completion_ratio model_price billing_mode billing_expr "
	for modelName, ratios := range payload.Differences {
		if validateMCPModelName(modelName) != nil {
			delete(payload.Differences, modelName)
			continue
		}
		for field, difference := range ratios {
			if !strings.Contains(allowed, " "+field+" ") {
				delete(ratios, field)
				continue
			}
			values := []json.RawMessage{difference.Current}
			for _, value := range difference.Upstreams {
				values = append(values, value)
			}
			for _, value := range values {
				kind := common.GetJsonType(value)
				if kind != "number" && kind != "string" && kind != "null" {
					return nil, true, errors.New("invalid pricing comparison value")
				}
				if kind == "string" {
					var text string
					_ = common.Unmarshal(value, &text)
					if _, hidden := mcpScrubConfigValue(text, ""); hidden {
						return nil, true, errors.New("unsafe pricing comparison value")
					}
				}
			}
		}
	}
	result, err := common.Marshal(map[string]any{"success": true, "data": payload})
	return result, true, err
}
