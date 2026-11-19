package service

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/QuantumNous/new-api/common"
)

var mcpMetadataCredential = regexp.MustCompile(`(?i)(bearer\s+|sk-[a-z0-9_-]{4,}|mcp_[a-z0-9_-]{8,}|(?:password|secret|access[_-]?token|api[_-]?key|authorization)\s*[=:])`)
var mcpMetadataURL = regexp.MustCompile(`https?://[^\s<>"')]+`)
var mcpMetadataEndpointName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var mcpModelCredential = regexp.MustCompile(`(?i)(?:^|[\s,;:=(])(?:bearer\s+|sk-[a-z0-9_-]{4,}|mcp_[a-z0-9_-]{8,})|(?:password|secret|access[_-]?token|api[_-]?key|authorization)\s*[=:]`)

// 完整覆盖更新依赖这些字段；只在对应元数据工具开放，不全局扩大自由文本范围。
func mcpEditableMetadataField(tool, field string) bool {
	var fields string
	switch tool {
	case "newapi_models_list", "newapi_models_search", "newapi_model_get", "newapi_model_create", "newapi_model_update":
		fields = " id model_name description icon tags vendor_id endpoints status sync_official name_rule created_time updated_time "
	case "newapi_vendors_list", "newapi_vendor_get", "newapi_vendor_create", "newapi_vendor_update":
		fields = " id name description icon status created_time updated_time "
	case "newapi_group_tags_list", "newapi_group_tag_create", "newapi_group_tag_update":
		fields = " id name description icons sort_order group_ids match_keywords "
	case "newapi_channel_get", "newapi_channels_list", "newapi_channels_search", "newapi_channel_update":
		fields = " models "
	}
	return fields != "" && strings.Contains(fields, " "+field+" ")
}

// 不截断或删除编辑必需字段：无法安全展示时返回整体错误，避免把不完整对象回写。
func validateMCPEditableMetadata(field string, raw json.RawMessage) error {
	if field == "model_name" {
		var name string
		if common.GetJsonType(raw) != "string" || common.Unmarshal(raw, &name) != nil {
			return errors.New("invalid model name type")
		}
		return validateMCPModelName(name)
	}
	if field == "models" {
		var csv string
		if common.GetJsonType(raw) != "string" || common.Unmarshal(raw, &csv) != nil || len(csv) > 8192 {
			return errors.New("unsafe or oversized channel model list")
		}
		names := strings.Split(csv, ",")
		if len(names) > 512 {
			return errors.New("too many channel models")
		}
		for _, name := range names {
			if err := validateMCPModelName(name); err != nil {
				return err
			}
		}
		return nil
	}
	if field == "endpoints" {
		var text string
		if common.Unmarshal(raw, &text) != nil || len(text) > 8192 {
			return errors.New("unsafe metadata endpoints")
		}
		if text == "" {
			return nil
		}
		var endpoints map[string]map[string]string
		if common.UnmarshalJsonStr(text, &endpoints) == nil && endpoints != nil {
			for name, endpoint := range endpoints {
				if !mcpMetadataEndpointName.MatchString(name) || len(endpoint) != 2 {
					return errors.New("unsafe metadata endpoints")
				}
				path, method := endpoint["path"], endpoint["method"]
				if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "?#%\\\r\n") || strings.Contains(path, "..") || len(path) > 256 || mcpMetadataCredential.MatchString(path) {
					return errors.New("unsafe metadata endpoints")
				}
				switch method {
				case "GET", "POST", "PUT", "PATCH", "DELETE":
				default:
					return errors.New("unsafe metadata endpoints")
				}
			}
			return nil
		}
		var names []string
		if common.UnmarshalJsonStr(text, &names) != nil || names == nil {
			return errors.New("unsafe metadata endpoints")
		}
		for _, name := range names {
			if !mcpMetadataEndpointName.MatchString(name) {
				return errors.New("unsafe metadata endpoints")
			}
		}
		return nil
	}
	if common.GetJsonType(raw) == "array" {
		var values []json.RawMessage
		if common.Unmarshal(raw, &values) != nil || len(values) > 10000 {
			return errors.New("metadata array too large")
		}
		for _, value := range values {
			if err := validateMCPEditableMetadata(field, value); err != nil {
				return err
			}
		}
		return nil
	}
	if common.GetJsonType(raw) == "object" {
		return errors.New("unexpected metadata object")
	}
	if common.GetJsonType(raw) != "string" {
		return nil
	}
	var text string
	if err := common.Unmarshal(raw, &text); err != nil {
		return err
	}
	limit := 2048
	if field == "description" {
		limit = 8192
	}
	if len(text) > limit || strings.ContainsRune(text, '\x00') || mcpMetadataCredential.MatchString(text) {
		return errors.New("metadata contains sensitive or oversized text")
	}
	for _, rawURL := range mcpMetadataURL.FindAllString(text, -1) {
		u, err := url.Parse(rawURL)
		if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return errors.New("metadata URL contains credentials or query data")
		}
	}
	if field == "icon" || field == "icons" {
		if strings.Contains(text, ":") && !strings.HasPrefix(text, "https://") && !strings.HasPrefix(text, "http://") {
			return errors.New("unsafe metadata icon URL")
		}
	}
	return nil
}

// 模型名称允许业务命名中的斜杠、冒号和 Unicode，但不能承载 URL、凭据或原始 JSON。
func validateMCPModelName(name string) error {
	trimmed := strings.TrimSpace(name)
	if len(name) > 512 || strings.Contains(name, "://") || strings.Contains(name, "\\") || strings.Contains(name, ",") || strings.ContainsAny(trimmed, "{}[]") || mcpModelCredential.MatchString(name) {
		return errors.New("unsafe or oversized model name")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return errors.New("invalid model name")
		}
	}
	return nil
}

// 关联数据只开放固定形状，不允许借模型列表携带任意对象或自由文本配置。
func projectMCPModelAssociation(field string, raw json.RawMessage) (json.RawMessage, error) {
	switch field {
	case "matched_models", "data":
		// 缺失列表在所有模型均已建档时可能由现有控制器返回 null。
		if field == "data" && common.GetJsonType(raw) == "null" {
			return raw, nil
		}
		var names []string
		if common.GetJsonType(raw) != "array" || common.Unmarshal(raw, &names) != nil {
			return nil, errors.New("invalid model names")
		}
		for _, name := range names {
			if err := validateMCPModelName(name); err != nil {
				return nil, err
			}
		}
		return raw, nil
	case "matched_count":
		var count int64
		if common.GetJsonType(raw) != "number" || common.Unmarshal(raw, &count) != nil || count < 0 || count > 1<<31-1 {
			return nil, errors.New("invalid matched model count")
		}
		return raw, nil
	case "quota_types":
		var quotaTypes []int
		if common.GetJsonType(raw) != "array" || common.Unmarshal(raw, &quotaTypes) != nil {
			return nil, errors.New("invalid model quota types")
		}
		for _, quotaType := range quotaTypes {
			if quotaType < 0 || quotaType > 1<<31-1 {
				return nil, errors.New("invalid model quota type")
			}
		}
		return raw, nil
	case "bound_channels":
		var channels []map[string]json.RawMessage
		if common.GetJsonType(raw) != "array" || common.Unmarshal(raw, &channels) != nil {
			return nil, errors.New("invalid model channel bindings")
		}
		result := make([]map[string]json.RawMessage, 0, len(channels))
		for _, channel := range channels {
			var name string
			var channelType int
			if common.GetJsonType(channel["name"]) != "string" || common.Unmarshal(channel["name"], &name) != nil || validateMCPEditableMetadata("name", channel["name"]) != nil || common.Unmarshal(channel["type"], &channelType) != nil || channelType < 1 || channelType > 10000 {
				return nil, errors.New("invalid bound channel metadata")
			}
			result = append(result, map[string]json.RawMessage{"name": channel["name"], "type": channel["type"]})
		}
		return common.Marshal(result)
	default:
		return nil, errors.New("unknown model association")
	}
}
