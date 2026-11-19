package service

import (
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// projectMCPFullChannelResponse 由主入口在确认业务成功后调用；不转发 message 和未知字段。
func projectMCPFullChannelResponse(tool MCPTool, raw []byte) ([]byte, bool, error) {
	if tool.Name == "newapi_channel_config_get" {
		projected, err := projectMCPChannelConfiguration(tool, raw)
		return projected, true, err
	}
	var dataShape any
	group := map[string]any{"id": "integer", "code": "text", "name": "text", "description": "text", "ratio": "number", "user_selectable": "boolean", "exclusive": "boolean", "single_user_concurrency_limit": "integer", "status": "integer", "created_time": "integer", "updated_time": "integer", "auto_enabled": "boolean", "auto_order": "integer"}
	autoGroup := map[string]any{"user_selectable": "boolean", "description": "text"}
	priority := map[string]any{"id": "integer", "previous": "integer", "priority": "integer"}
	modelChanges := map[string]any{"id": "integer", "channel_id": "integer", "channel_name": "text", "added_models": []any{"model"}, "removed_models": []any{"model"}, "ignored_models": []any{"model"}, "remaining_models": []any{"model"}, "remaining_remove_models": []any{"model"}, "models": "csv"}
	switch tool.Name {
	case "newapi_channels_create_batch":
		// 宿主只返回成功状态；未来新增字段也不能将密钥或配置纳入回执。
		return []byte(`{"success":true}`), true, nil
	case "newapi_channel_codex_refresh":
		dataShape = map[string]any{"expires_at": "text", "last_refresh": "text", "channel_id": "integer", "channel_type": "integer", "channel_name": "text"}
	case "newapi_channel_codex_usage_reset":
		dataShape = map[string]any{"windows_reset": "integer"}
	case "newapi_channel_codex_usage":
		window := map[string]any{"used_percent": "number", "reset_at": "integer", "reset_after_seconds": "integer", "limit_window_seconds": "integer"}
		rateLimit := map[string]any{"plan_type": "text", "allowed": "boolean", "limit_reached": "boolean", "primary_window": window, "secondary_window": window}
		dataShape = map[string]any{"plan_type": "text", "rate_limit": rateLimit, "additional_rate_limits": []any{map[string]any{"limit_name": "text", "metered_feature": "text", "rate_limit": rateLimit, "primary_window": window, "secondary_window": window, "plan_type": "text"}}, "rate_limit_reset_credits": map[string]any{"available_count": "integer"}, "credits": map[string]any{"overage_limit_reached": "boolean"}, "spend_control": map[string]any{"reached": "boolean"}}
	case "newapi_channel_codex_reset_credits":
		dataShape = map[string]any{"available_count": "integer", "total_earned_count": "integer", "credits": []any{map[string]any{"reset_type": "text", "status": "text", "granted_at": "text", "expires_at": "text", "redeem_started_at": "text", "redeemed_at": "text"}}}
	case "newapi_vendors_search":
		dataShape = map[string]any{"total": "integer", "page": "integer", "page_size": "integer", "items": []any{map[string]any{"id": "integer", "name": "text", "description": "text", "icon": "text", "status": "integer", "created_time": "integer", "updated_time": "integer"}}}
	case "newapi_channel_priority", "newapi_channel_priority_reorder", "newapi_channel_priority_restore":
		dataShape = map[string]any{"revision": "revision", "items": []any{priority}, "changes": []any{priority}}
	case "newapi_channel_keys_status":
		dataShape = map[string]any{"keys": []any{map[string]any{"index": "integer", "status": "integer", "disabled_time": "integer"}}, "total": "integer", "page": "integer", "page_size": "integer", "total_pages": "integer", "enabled_count": "integer", "manual_disabled_count": "integer", "auto_disabled_count": "integer"}
	case "newapi_channel_models_fetch", "newapi_channel_models_fetch_preview":
		dataShape = []any{"model"}
	case "newapi_channel_tag_models":
		dataShape = "csv"
	case "newapi_channel_models_detect":
		dataShape = map[string]any{"channel_id": "integer", "channel_name": "text", "add_models": []any{"model"}, "remove_models": []any{"model"}, "last_check_time": "integer", "auto_added_models": "integer"}
	case "newapi_channel_models_apply":
		dataShape = modelChanges
	case "newapi_channels_models_apply":
		dataShape = map[string]any{"processed_channels": "integer", "added_models": "integer", "removed_models": "integer", "failed_channel_ids": []any{"integer"}, "results": []any{modelChanges}}
	case "newapi_models_sync":
		dataShape = map[string]any{"created_models": "integer", "created_vendors": "integer", "updated_models": "integer", "skipped_models": []any{"model"}, "created_list": []any{"model"}, "updated_list": []any{"model"}, "source": map[string]any{"locale": "text"}}
	case "newapi_models_sync_preview":
		// 冲突值可能是任意描述文本或远端数据，只公开字段名与模型名供选择覆盖。
		dataShape = map[string]any{"missing": []any{"model"}, "conflicts": []any{map[string]any{"model_name": "model", "fields": []any{map[string]any{"field": "sync_field"}}}}, "source": map[string]any{"locale": "text"}}
	case "newapi_groups_list", "newapi_groups_save":
		dataShape = []any{group}
	case "newapi_groups_code_migration_preview", "newapi_groups_code_migrate":
		dataShape = map[string]any{"can_execute": "boolean", "executed": "boolean", "groups": []any{map[string]any{"group_id": "integer", "name": "text", "old_code": "text", "target_code": "text"}}, "affected_channels": "integer", "affected_tokens": "integer", "affected_users": "integer", "affected_abilities": "integer", "affected_subscription_plans": "integer", "affected_subscriptions": "integer", "affected_options": "integer", "cache_invalidated": "integer", "cache_invalidation_failed": "integer"}
	case "newapi_groups_token_migration_preview", "newapi_groups_tokens_migrate":
		groupRef := map[string]any{"id": "integer", "code": "text", "name": "text"}
		dataShape = map[string]any{"source_group": groupRef, "target_group": groupRef, "target_group_mode": "text", "migrated_tokens": "integer", "deduplicated_tokens": "integer", "single_group_tokens": "integer", "multi_group_tokens": "integer", "affected_users": "integer", "cleaned_deleted_tokens": "integer", "cache_invalidated": "integer", "cache_invalidation_failed": "integer"}
	case "newapi_prefill_groups", "newapi_prefill_group_create", "newapi_prefill_group_update":
		prefill := map[string]any{"id": "integer", "name": "text", "type": "text", "items": []any{"model"}, "description": "text", "created_time": "integer", "updated_time": "integer"}
		dataShape = prefill
		if tool.Name == "newapi_prefill_groups" {
			dataShape = []any{prefill}
		}
	case "newapi_channel_config_update":
		// 配置只用作输入。输出固定路由摘要，避免将设置中嵌入的凭据重新暴露。
		dataShape = map[string]any{"id": "integer", "name": "text", "type": "integer", "vendor_id": "integer", "models": "csv", "group": "text", "group_ids": []any{"integer"}, "group_details": []any{map[string]any{"id": "integer", "code": "text", "name": "text"}}, "priority": "integer", "weight": "integer", "concurrency_limit": "integer", "status": "integer"}
	default:
		return nil, false, nil
	}
	if len(raw) > mcpResponseLimit(tool) {
		return nil, true, errors.New("response too large")
	}
	root := map[string]any{"success": "boolean", "data": dataShape}
	if tool.Name == "newapi_groups_list" || tool.Name == "newapi_groups_save" {
		root["auto_group"] = autoGroup
	}
	if tool.Name == "newapi_channel_codex_usage" || tool.Name == "newapi_channel_codex_usage_reset" || tool.Name == "newapi_channel_codex_reset_credits" {
		root["upstream_status"] = "integer"
	}
	result, err := projectMCPChannelShape(raw, root, 0)
	return result, true, err
}

// 配置读取只接受渠道编辑字段。含凭据的字段整体隐藏，避免返回可误保存的残缺 JSON。
func projectMCPChannelConfiguration(tool MCPTool, raw []byte) ([]byte, error) {
	if len(raw) > mcpResponseLimit(tool) {
		return nil, errors.New("response too large")
	}
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if common.Unmarshal(raw, &envelope) != nil || envelope.Data == nil {
		return nil, errors.New("invalid channel configuration")
	}
	shape := map[string]any{"id": "integer", "type": "integer", "vendor_id": "integer", "name": "text", "models": "csv", "group": "text", "group_ids": []any{"integer"}, "priority": "integer", "weight": "integer", "concurrency_limit": "integer", "auto_ban": "integer", "tag": "text", "remark": "text", "test_model": "model", "group_details": []any{map[string]any{"id": "integer", "code": "text", "name": "text"}}}
	dataRaw, err := common.Marshal(envelope.Data)
	if err != nil {
		return nil, err
	}
	projected, err := projectMCPChannelShape(dataRaw, shape, 0)
	if err != nil {
		return nil, err
	}
	var result map[string]json.RawMessage
	if err := common.Unmarshal(projected, &result); err != nil {
		return nil, err
	}
	redacted := []string{"key"}
	for _, field := range []string{"base_url", "openai_organization", "other", "setting", "settings", "model_mapping", "status_code_mapping", "param_override", "header_override"} {
		value, exists := envelope.Data[field]
		if !exists || common.GetJsonType(value) == "null" {
			continue
		}
		var text string
		if common.Unmarshal(value, &text) != nil || len(text) > 32768 {
			return nil, errors.New("invalid channel configuration field")
		}
		var parsed any = text
		if text != "" {
			_ = common.UnmarshalJsonStr(text, &parsed)
		}
		if field == "header_override" && strings.TrimSpace(text) != "" && strings.TrimSpace(text) != "{}" || mcpChannelConfigHasSecret(parsed, "", 0) {
			result[field] = json.RawMessage(`"[redacted]"`)
			redacted = append(redacted, field)
		} else {
			result[field] = value
		}
	}
	// 渠道详情不返回 key，因此只能声明隐藏状态，不能猜测凭据是否已配置。
	result["redacted_fields"], err = common.Marshal(redacted)
	if err != nil {
		return nil, err
	}
	return common.Marshal(map[string]any{"success": true, "data": result})
}

func mcpChannelConfigHasSecret(value any, field string, depth int) bool {
	if depth > 16 {
		return true
	}
	// 插件标识和密钥类型是业务枚举，不是凭据内容。
	if field != "task_plugin_key" && field != "vertex_key_type" && field != "aws_key_type" && (mcpSecretSettingName.MatchString(field) || strings.Contains(strings.ToLower(field), "auth")) {
		return true
	}
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			if mcpChannelConfigHasSecret(child, key, depth+1) {
				return true
			}
		}
	case []any:
		for _, child := range item {
			if mcpChannelConfigHasSecret(child, "", depth+1) {
				return true
			}
		}
	case string:
		if strings.Contains(item, "[redacted]") || mcpModelCredential.MatchString(item) {
			return true
		}
		if strings.Contains(item, "://") {
			address, err := url.Parse(item)
			if err != nil || address.User != nil || address.RawQuery != "" || address.Fragment != "" {
				return true
			}
		}
		var nested any
		if common.UnmarshalJsonStr(item, &nested) == nil {
			switch nested.(type) {
			case map[string]any, []any:
				return mcpChannelConfigHasSecret(nested, "", depth+1)
			}
		}
	}
	return false
}

var mcpChannelRevisionPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// projectMCPChannelShape 只遍历声明的形状；不会按键名递归放行任意业务对象。
func projectMCPChannelShape(raw json.RawMessage, shape any, depth int) (json.RawMessage, error) {
	if depth > 16 {
		return nil, errors.New("channel response nesting too deep")
	}
	if common.GetJsonType(raw) == "null" && depth > 0 {
		return raw, nil
	}
	switch expected := shape.(type) {
	case map[string]any:
		var object map[string]json.RawMessage
		if common.GetJsonType(raw) != "object" || common.Unmarshal(raw, &object) != nil {
			return nil, errors.New("invalid channel response object")
		}
		out := map[string]json.RawMessage{}
		for field, childShape := range expected {
			value, ok := object[field]
			if !ok {
				continue
			}
			child, err := projectMCPChannelShape(value, childShape, depth+1)
			if err != nil {
				return nil, err
			}
			out[field] = child
		}
		return common.Marshal(out)
	case []any:
		var array []json.RawMessage
		if common.GetJsonType(raw) != "array" || common.Unmarshal(raw, &array) != nil || len(array) > 10000 {
			return nil, errors.New("invalid channel response array")
		}
		out := make([]json.RawMessage, 0, len(array))
		for _, value := range array {
			child, err := projectMCPChannelShape(value, expected[0], depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, child)
		}
		return common.Marshal(out)
	case string:
		switch expected {
		case "integer", "number":
			var number float64
			if common.GetJsonType(raw) != "number" || common.Unmarshal(raw, &number) != nil || math.IsInf(number, 0) || math.IsNaN(number) || math.Abs(number) > 9007199254740991 || expected == "integer" && math.Trunc(number) != number {
				return nil, errors.New("invalid channel response number")
			}
		case "boolean":
			var value bool
			if common.GetJsonType(raw) != "boolean" || common.Unmarshal(raw, &value) != nil {
				return nil, errors.New("invalid channel response boolean")
			}
		default:
			var text string
			if common.GetJsonType(raw) != "string" || common.Unmarshal(raw, &text) != nil {
				return nil, errors.New("invalid channel response text")
			}
			switch expected {
			case "model":
				if err := validateMCPModelName(text); err != nil {
					return nil, err
				}
			case "csv":
				if err := validateMCPEditableMetadata("models", raw); err != nil {
					return nil, err
				}
			case "revision":
				if !mcpChannelRevisionPattern.MatchString(text) {
					return nil, errors.New("invalid channel priority revision")
				}
			case "sync_field":
				switch text {
				case "description", "icon", "tags", "vendor", "name_rule", "status":
				default:
					return nil, errors.New("invalid model sync field")
				}
			case "text":
				if err := validateMCPEditableMetadata("description", raw); err != nil {
					return nil, err
				}
			default:
				return nil, errors.New("unknown channel projection type")
			}
		}
		return raw, nil
	default:
		return nil, errors.New("invalid channel projection shape")
	}
}
