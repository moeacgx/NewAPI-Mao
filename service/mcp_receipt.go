package service

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// 扩展写工具的回执只保存稳定身份、版本与计数，不保存编辑正文或配置内容。
// 首次结果仍返回调用方；重试通过这些身份查询当前资源，不再次执行业务。
func mcpManagementReceipt(raw []byte, requestID string, success bool, transient bool, toolName ...string) (MCPToolResult, error) {
	data := json.RawMessage(`{}`)
	var err error
	if !transient {
		name := ""
		if len(toolName) > 0 {
			name = toolName[0]
		}
		data, err = mcpReceiptMetadata(raw, "", "", name, 0)
		if err != nil {
			return MCPToolResult{}, err
		}
	}
	payload, err := common.Marshal(map[string]any{
		"success": success, "request_id": requestID, "result_not_retained": true,
		"receipt": data,
		"message": "Operation completed; use receipt identifiers to read current state. Do not repeat the operation",
	})
	return MCPToolResult{Content: []MCPTextContent{{Type: "text", Text: string(payload)}}, IsError: !success}, err
}

func mcpReceiptMetadata(raw json.RawMessage, field, path, toolName string, depth int) (json.RawMessage, error) {
	if depth > 16 {
		return nil, errors.New("receipt nesting too deep")
	}
	switch common.GetJsonType(raw) {
	case "object":
		var object map[string]json.RawMessage
		if err := common.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		result := map[string]json.RawMessage{}
		for key, value := range object {
			pluginIdentity := (toolName == "newapi_plugin_upload" || toolName == "newapi_plugin_upload_put") && path == "data" && key == "key"
			if !pluginIdentity && !strings.Contains(" success data items results tasks module modules id task_id store_id product_id deployment_id container_id channel_id user_id plan_id subscription_id bot_id version config_version status count total created_count updated_count deleted_count deleted_events deleted_jobs success_count failure_count ", " "+key+" ") {
				continue
			}
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			child, err := mcpReceiptMetadata(value, key, childPath, toolName, depth+1)
			if err != nil {
				return nil, err
			}
			if child != nil {
				result[key] = child
			}
		}
		return common.Marshal(result)
	case "array":
		var values []json.RawMessage
		if err := common.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		result := make([]json.RawMessage, 0, len(values))
		for _, value := range values {
			child, err := mcpReceiptMetadata(value, field, path, toolName, depth+1)
			if err != nil {
				return nil, err
			}
			if child != nil {
				result = append(result, child)
			}
		}
		return common.Marshal(result)
	case "string":
		pluginIdentity := (toolName == "newapi_plugin_upload" || toolName == "newapi_plugin_upload_put") && path == "data.key"
		if !pluginIdentity && !strings.Contains(" id task_id store_id product_id deployment_id container_id version status ", " "+field+" ") {
			return nil, nil
		}
		var value string
		if common.Unmarshal(raw, &value) != nil || len(value) > 128 || !mcpTypedPathSegment.MatchString(value) || mcpModelCredential.MatchString(value) {
			return nil, nil
		}
		return raw, nil
	case "number", "boolean", "null":
		return raw, nil
	default:
		return nil, nil
	}
}
