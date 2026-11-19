package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

const MCPMaxBodyBytes = 64 * 1024
const MCPMaxResponseBytes = 32 * 1024
const MCPMaxTransportBytes = 1024 * 1024

// 大型配置与归档正文仍有明确内存上限；原有摘要工具保留 32 KiB 限制。
func mcpResponseLimit(tool MCPTool) int {
	if tool.TypedArguments {
		return 4 * 1024 * 1024
	}
	return MCPMaxResponseBytes
}

type MCPArguments struct {
	Path      map[string]string `json:"path,omitempty"`
	Query     map[string]string `json:"query,omitempty"`
	Body      json.RawMessage   `json:"body,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

var mcpPathSegment = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var mcpTypedPathSegment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+@:-]*$`)

// PrepareMCPRequest 只从可信目录构造相对 API 路径，不接收用户的 URL 或方法。
func PrepareMCPRequest(tool MCPTool, raw []byte) (*MCPArguments, string, []byte, error) {
	if len(raw) == 0 {
		raw = []byte(`{}`)
	}
	limit := MCPMaxBodyBytes
	if tool.TypedArguments {
		limit = MCPMaxTransportBytes
	}
	if tool.UploadField != "" {
		limit = MCPMaxUploadBytes
	}
	if len(raw) > limit {
		return nil, "", nil, errors.New("arguments too large")
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return nil, "", nil, errors.New("invalid arguments JSON")
	}
	allowUnboundModelVendor := tool.Name == "newapi_model_create" || tool.Name == "newapi_model_update"
	if !tool.TypedArguments {
		if err := validateMCPStructure(value, 0, allowUnboundModelVendor); err != nil {
			return nil, "", nil, err
		}
	}
	// 将目录里的 []string / []int 等统一为 JSON Schema 的标准数据形状。
	schemaJSON, err := common.Marshal(tool.InputSchema)
	if err != nil {
		return nil, "", nil, errors.New("invalid tool schema")
	}
	var schema map[string]any
	if err = common.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, "", nil, err
	}
	if err = validateMCPSchema(schema, value, 0); err != nil {
		return nil, "", nil, err
	}
	var arguments MCPArguments
	if err = common.Unmarshal(raw, &arguments); err != nil {
		return nil, "", nil, errors.New("invalid argument types")
	}
	if tool.Write && (!mcpPathSegment.MatchString(arguments.RequestID) || len(arguments.RequestID) > 128) {
		return nil, "", nil, errors.New("request_id must be 1-128 letters, digits, underscores or hyphens")
	}
	if tool.Path == "" && tool.Name == "newapi_context" {
		return &arguments, "", nil, nil
	}
	if !strings.HasPrefix(tool.Path, "/api/") || strings.ContainsAny(tool.Path, "?#%\\") || strings.Contains(tool.Path, "..") || strings.HasPrefix(tool.Path, "/api/mcp") {
		return nil, "", nil, errors.New("invalid registered route")
	}
	switch tool.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return nil, "", nil, errors.New("invalid registered method")
	}
	segments := strings.Split(tool.Path, "/")
	for index, segment := range segments {
		if !strings.HasPrefix(segment, ":") {
			continue
		}
		name := strings.TrimPrefix(segment, ":")
		v := arguments.Path[name]
		validPath := mcpPathSegment.MatchString(v)
		if tool.TypedArguments {
			validPath = mcpTypedPathSegment.MatchString(v) && v != "." && v != ".."
		}
		if len(v) > 128 || !validPath {
			return nil, "", nil, errors.New("invalid path parameter")
		}
		stringID := false
		for _, field := range tool.StringPathIDs {
			if field == name {
				stringID = true
				break
			}
		}
		if !stringID && (name == "id" || strings.HasSuffix(name, "_id") && name != "task_id") {
			id, parseErr := strconv.ParseInt(v, 10, 32)
			if parseErr != nil || id <= 0 {
				return nil, "", nil, errors.New("invalid numeric ID")
			}
		}
		segments[index] = url.PathEscape(v)
	}
	query := url.Values{}
	for key, v := range arguments.Query {
		if !mcpPathSegment.MatchString(key) || len(key) > 64 || strings.ContainsAny(v, "\x00\r\n") {
			return nil, "", nil, errors.New("invalid query parameter")
		}
		switch key {
		case "p", "page", "page_size", "ps", "limit", "size", "offset":
			n, parseErr := strconv.ParseInt(v, 10, 32)
			minimum, maximum := int64(1), int64(10000)
			if key == "page_size" || key == "ps" || key == "limit" || key == "size" {
				maximum = 100
			}
			if key == "offset" {
				minimum = 0
				maximum = 1000000
			}
			if parseErr != nil || n < minimum || n > maximum {
				return nil, "", nil, errors.New("pagination out of range")
			}
		}
		stringID := false
		for _, field := range tool.StringQueryIDs {
			stringID = stringID || field == key
		}
		if !stringID && (key == "id" || strings.HasSuffix(key, "_id") && key != "task_id" && key != "request_id") {
			id, parseErr := strconv.ParseInt(v, 10, 32)
			if parseErr != nil || id <= 0 {
				return nil, "", nil, errors.New("invalid query ID")
			}
		}
		query.Set(key, v)
	}
	if tool.Name == "newapi_log_cleanup_start" {
		cutoff, parseErr := strconv.ParseInt(arguments.Query["target_timestamp"], 10, 64)
		if parseErr != nil || cutoff <= 0 || cutoff > time.Now().Unix() {
			return nil, "", nil, errors.New("cleanup timestamp must be in the past")
		}
	}
	if tool.Name == "newapi_channel_affinity_clear" && arguments.Query["all"] != "true" && strings.TrimSpace(arguments.Query["rule_name"]) == "" {
		return nil, "", nil, errors.New("all=true or rule_name is required")
	}
	path := strings.Join(segments, "/")
	if len(query) != 0 {
		path += "?" + query.Encode()
	}
	// 保留请求体原始数值文本，不能经 float64 中转丢失业务 ID 的精度。
	body := []byte(arguments.Body)
	bodyArray := false
	if properties, ok := tool.InputSchema["properties"].(map[string]any); ok && tool.TypedArguments {
		if bodySchema, ok := properties["body"].(map[string]any); ok {
			bodyArray = bodySchema["type"] == "array"
		}
	}
	validBody := bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) || bodyArray && bytes.HasPrefix(bytes.TrimSpace(body), []byte("["))
	if len(body) != 0 && !validBody {
		return nil, "", nil, errors.New("body must be an object")
	}
	if len(body) != 0 && !bodyArray {
		var fields map[string]json.RawMessage
		if err := common.Unmarshal(body, &fields); err != nil {
			return nil, "", nil, errors.New("invalid body JSON")
		}
		// 在申请写请求执行权和调用真实路由之前校验，防止成功写入后才因响应不安全而失败。
		for field, raw := range fields {
			var validationErr error
			switch {
			case tool.Name == "newapi_channel_update" && field == "models":
				validationErr = validateMCPChannelModelsForWrite(raw)
			case mcpEditableMetadataField(tool.Name, field):
				validationErr = validateMCPEditableMetadata(field, raw)
			}
			if validationErr != nil {
				return nil, "", nil, validationErr
			}
		}
		if tool.Name == "newapi_channel_create" {
			var channel map[string]json.RawMessage
			if common.Unmarshal(fields["channel"], &channel) != nil || channel == nil {
				return nil, "", nil, errors.New("invalid channel body")
			}
			if err := validateMCPChannelModelsForWrite(channel["models"]); err != nil {
				return nil, "", nil, err
			}
		}
		if tool.Name == "newapi_setting_update" {
			var optionKey, optionValue string
			_ = common.Unmarshal(fields["key"], &optionKey)
			_ = common.Unmarshal(fields["value"], &optionValue)
			common.OptionMapRWMutex.RLock()
			_, known := common.OptionMap[optionKey]
			common.OptionMapRWMutex.RUnlock()
			if !known || strings.HasPrefix(optionKey, "payment_setting.compliance_") {
				return nil, "", nil, errors.New("unknown or protected option")
			}
			if strings.Contains(optionValue, "[redacted]") {
				return nil, "", nil, errors.New("redacted placeholders cannot be saved; supply the intended configuration")
			}
			if err := validateMCPOptionValue(optionKey, optionValue); err != nil {
				return nil, "", nil, err
			}
		}
		if tool.Name == "newapi_ratio_fetch" {
			if err := validateMCPRatioRequest(body); err != nil {
				return nil, "", nil, err
			}
		}
	}
	if tool.UploadField != "" {
		if _, _, err := mcpUploadBody(tool.UploadField, body); err != nil {
			return nil, "", nil, err
		}
	}
	if tool.TypedArguments && tool.Category == "channels" || tool.TypedArguments && tool.Category == "groups" {
		if err := validateMCPFullChannelArguments(tool, body); err != nil {
			return nil, "", nil, err
		}
	}
	return &arguments, path, body, nil
}

// 写入与宿主的模型名称长度限制一致；读取保留较长历史名称以便诊断。
func validateMCPChannelModelsForWrite(raw json.RawMessage) error {
	if err := validateMCPEditableMetadata("models", raw); err != nil {
		return err
	}
	var csv string
	if err := common.Unmarshal(raw, &csv); err != nil {
		return err
	}
	for _, name := range strings.Split(csv, ",") {
		if strings.TrimSpace(name) == "" {
			return errors.New("channel model list must not contain empty names")
		}
		if len(name) > 255 {
			return errors.New("channel model name exceeds 255 bytes")
		}
	}
	return nil
}

// 即使业务 body 允许额外字段，仍执行结构深度和 ID 范围约束。
func validateMCPStructure(value any, depth int, allowUnboundModelVendor bool) error {
	if depth > 32 {
		return errors.New("arguments nesting too deep")
	}
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			lower := strings.ToLower(key)
			if lower == "id" || strings.HasSuffix(lower, "_id") {
				if number, ok := child.(float64); ok {
					// 只允许模型元数据 body 顶层 vendor_id 的“未绑定”值，不扩大其它 ID 或嵌套对象。
					unbound := allowUnboundModelVendor && depth == 1 && key == "vendor_id" && number == 0
					if !unbound && (number < 1 || number > float64(1<<31-1) || math.Trunc(number) != number) {
						return errors.New("ID out of range")
					}
				}
			}
			if lower == "ids" || strings.HasSuffix(lower, "_ids") {
				if ids, ok := child.([]any); ok {
					for _, id := range ids {
						number, ok := id.(float64)
						if !ok || number < 1 || number > float64(1<<31-1) || math.Trunc(number) != number {
							return errors.New("invalid ID list")
						}
					}
				}
			}
			allowChildVendor := allowUnboundModelVendor && depth == 0 && key == "body"
			if err := validateMCPStructure(child, depth+1, allowChildVendor); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range item {
			if err := validateMCPStructure(child, depth+1, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// 目录使用的 JSON Schema 子集在这里逐项执行；新增关键字默认拒绝，避免只展示约束。
func validateMCPSchema(schema map[string]any, value any, depth int) error {
	if depth > 32 {
		return errors.New("arguments nesting too deep")
	}
	for keyword := range schema {
		switch keyword {
		case "type", "properties", "required", "additionalProperties", "maxProperties", "minProperties", "minLength", "maxLength", "pattern", "enum", "minimum", "maximum", "items", "minItems", "maxItems", "description", "title", "default":
		default:
			return errors.New("unsupported schema constraint")
		}
	}
	if choices, ok := schema["enum"].([]any); ok {
		actual, _ := common.Marshal(value)
		found := false
		for _, choice := range choices {
			expected, _ := common.Marshal(choice)
			found = found || bytes.Equal(actual, expected)
		}
		if !found {
			return errors.New("value outside allowed enum")
		}
	}
	if alternatives, ok := schema["type"].([]any); ok {
		if len(alternatives) == 0 || len(alternatives) > 7 {
			return errors.New("invalid schema type alternatives")
		}
		for _, alternative := range alternatives {
			kind, ok := alternative.(string)
			if !ok {
				return errors.New("invalid schema type")
			}
			branch := make(map[string]any, len(schema))
			for key, entry := range schema {
				branch[key] = entry
			}
			branch["type"] = kind
			if validateMCPSchema(branch, value, depth+1) == nil {
				return nil
			}
		}
		return errors.New("value does not match allowed types")
	}
	switch schema["type"] {
	case "null":
		if value != nil {
			return errors.New("expected null")
		}
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			return errors.New("expected object")
		}
		if max, ok := schema["maxProperties"].(float64); ok && float64(len(object)) > max {
			return errors.New("too many properties")
		}
		if min, ok := schema["minProperties"].(float64); ok && float64(len(object)) < min {
			return errors.New("too few properties")
		}
		if required, ok := schema["required"].([]any); ok {
			for _, item := range required {
				key, ok := item.(string)
				if !ok {
					return errors.New("invalid schema")
				}
				if _, exists := object[key]; !exists {
					return errors.New("required property missing")
				}
			}
		}
		properties, _ := schema["properties"].(map[string]any)
		for key, item := range object {
			if child, ok := properties[key].(map[string]any); ok {
				if err := validateMCPSchema(child, item, depth+1); err != nil {
					return err
				}
				continue
			}
			switch additional := schema["additionalProperties"].(type) {
			case bool:
				if !additional {
					return errors.New("unknown argument property")
				}
			case map[string]any:
				if err := validateMCPSchema(additional, item, depth+1); err != nil {
					return err
				}
			}
		}
	case "string":
		text, ok := value.(string)
		if !ok {
			return errors.New("expected string")
		}
		length := float64(utf8.RuneCountInString(text))
		if min, ok := schema["minLength"].(float64); ok && length < min {
			return errors.New("string too short")
		}
		if max, ok := schema["maxLength"].(float64); ok && length > max {
			return errors.New("string too long")
		}
		if pattern, ok := schema["pattern"].(string); ok {
			matched, err := regexp.MatchString(pattern, text)
			if err != nil || !matched {
				return errors.New("string does not match pattern")
			}
		}
	case "number", "integer":
		number, ok := value.(float64)
		if !ok || math.IsInf(number, 0) || math.IsNaN(number) {
			return errors.New("expected number")
		}
		if schema["type"] == "integer" && math.Trunc(number) != number {
			return errors.New("expected integer")
		}
		if min, ok := schema["minimum"].(float64); ok && number < min {
			return errors.New("number too small")
		}
		if max, ok := schema["maximum"].(float64); ok && number > max {
			return errors.New("number too large")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("expected boolean")
		}
	case "array":
		items, ok := value.([]any)
		if !ok {
			return errors.New("expected array")
		}
		if min, ok := schema["minItems"].(float64); ok && float64(len(items)) < min {
			return errors.New("too few items")
		}
		if max, ok := schema["maxItems"].(float64); ok && float64(len(items)) > max {
			return errors.New("too many items")
		}
		if child, ok := schema["items"].(map[string]any); ok {
			for _, item := range items {
				if err := validateMCPSchema(child, item, depth+1); err != nil {
					return err
				}
			}
		}
	default:
		return errors.New("unsupported schema type")
	}
	return nil
}
