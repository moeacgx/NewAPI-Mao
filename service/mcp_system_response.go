package service

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

var mcpSecretSettingName = regexp.MustCompile(`(?i)(password|passwd|secret|credential|private.?key|api.?key|access.?key|access.?token|refresh.?token|client.?token|bot.?token|authorization|cookie|token$|key$)`)

// MCPOptionView 只用于私有 MCP 配置读取；敏感值显式标记，不能误当成可回写的空值。
type MCPOptionView struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Redacted bool   `json:"redacted"`
}

func MCPProjectOption(key, value string) MCPOptionView {
	result := MCPOptionView{Key: key, Value: value}
	if mcpSecretSettingName.MatchString(key) {
		result.Value, result.Redacted = "", true
		return result
	}
	var parsed any
	if common.UnmarshalJsonStr(value, &parsed) == nil {
		clean, redacted := mcpScrubConfigAt(parsed, "", key, "")
		if redacted {
			encoded, err := common.Marshal(clean)
			if err != nil {
				result.Value, result.Redacted = "", true
			} else {
				result.Value, result.Redacted = string(encoded), true
			}
		}
		return result
	}
	_, result.Redacted = mcpScrubConfigValue(value, "")
	if result.Redacted {
		result.Value = ""
	}
	return result
}

func mcpScrubConfigValue(value any, field string) (any, bool) {
	return mcpScrubConfigAt(value, field, "", "")
}

func mcpScrubConfigAt(value any, field, option, path string) (any, bool) {
	// 亲和规则的 key 是字段名称，不是字段内容；仅豁免宿主明确的结构位置。
	businessKey := option == "channel_affinity_setting" && path == "rules[].key_sources[].key" ||
		option == "channel_affinity_setting.rules" && path == "[].key_sources[].key"
	if mcpSecretSettingName.MatchString(field) && !businessKey {
		return "[redacted]", true
	}
	switch typed := value.(type) {
	case map[string]any:
		result, redacted := map[string]any{}, false
		for key, child := range typed {
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			clean, hidden := mcpScrubConfigAt(child, key, option, childPath)
			result[key] = clean
			redacted = redacted || hidden
		}
		return result, redacted
	case []any:
		result, redacted := make([]any, len(typed)), false
		for i, child := range typed {
			clean, hidden := mcpScrubConfigAt(child, "", option, path+"[]")
			result[i] = clean
			redacted = redacted || hidden
		}
		return result, redacted
	case string:
		if mcpModelCredential.MatchString(typed) {
			return "[redacted]", true
		}
		for _, text := range mcpMetadataURL.FindAllString(typed, -1) {
			address, err := url.Parse(text)
			if err != nil || address.User != nil {
				return "[redacted]", true
			}
			for key := range address.Query() {
				if mcpSecretSettingName.MatchString(key) {
					return "[redacted]", true
				}
			}
		}
	}
	return value, false
}

// 明确选择公开 DTO 可保留编辑所需字段，同时丢弃未来未声明的响应字段。
func projectMCPFullSystemResponse(tool MCPTool, raw []byte) ([]byte, bool, error) {
	var target any
	switch tool.Name {
	case "newapi_security_config", "newapi_security_config_update":
		target = &PromptAuditConfig{}
	case "newapi_security_policy", "newapi_security_policy_update":
		target = &SecurityAuditBuiltinPolicy{}
	case "newapi_archive_config", "newapi_archive_config_update":
		target = &RequestArchiveConfig{}
	case "newapi_conversation_config", "newapi_conversation_config_update":
		target = &ConversationArchiveConfigView{}
	case "newapi_model_guard_config", "newapi_model_guard_update":
		target = &UpstreamModelGuardConfig{}
	case "newapi_settings_list":
		target = &struct {
			Items    []MCPOptionView `json:"items"`
			Total    int             `json:"total"`
			Page     int             `json:"page"`
			PageSize int             `json:"page_size"`
		}{}
	case "newapi_security_delete_preview":
		target = &PromptAuditDeletePreview{}
	case "newapi_update_release":
		target = &SelfUpdateReleaseInfo{}
	case "newapi_channel_affinity_cache":
		target = &ChannelAffinityCacheStats{}
	case "newapi_channel_affinity_usage":
		target = &ChannelAffinityUsageCacheStats{}
	}
	if target != nil {
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if err := common.Unmarshal(raw, &envelope); err != nil {
			return nil, true, err
		}
		if err := common.Unmarshal(envelope.Data, target); err != nil {
			return nil, true, err
		}
		projected, err := common.Marshal(map[string]any{"success": true, "data": target})
		if err == nil && (tool.Name == "newapi_security_config" || tool.Name == "newapi_security_config_update" ||
			tool.Name == "newapi_security_policy" || tool.Name == "newapi_security_policy_update" ||
			tool.Name == "newapi_archive_config" || tool.Name == "newapi_archive_config_update" ||
			tool.Name == "newapi_conversation_config" || tool.Name == "newapi_conversation_config_update" ||
			tool.Name == "newapi_model_guard_config" || tool.Name == "newapi_model_guard_update" || tool.Name == "newapi_channel_affinity_usage") {
			var names map[string]string
			projected, err = mcpSystemGroupNames(projected, &names, 0)
		}
		return projected, true, err
	}
	fields := ""
	switch tool.Name {
	case "newapi_self_update":
		fields = "current_version target_version asset_name restart_scheduled exit_delay_seconds already_up_to_date downloaded_bytes downloaded_sha256 expected_sha256 running_in_docker allow_non_docker_run"
	case "newapi_health_check":
		fields = "http_stats active_connections"
	case "newapi_performance_logs":
		fields = "enabled file_count total_size oldest_time newest_time files name size mod_time"
	case "newapi_performance_logs_cleanup", "newapi_instances_cleanup", "newapi_instance_delete", "newapi_conversations_clear", "newapi_channel_affinity_clear":
		fields = "deleted deleted_count deleted_files freed_bytes freed_size remaining_files"
	case "newapi_security_delete_preview":
		fields = "matched_events matched_jobs total_events total_jobs event_count job_count confirmation_token expires_at filter_hash cutoff_id max_event_id"
	case "newapi_security_event_delete", "newapi_security_events_delete", "newapi_security_delete_filtered":
		fields = "deleted_events deleted_jobs"
	case "newapi_security_probe", "newapi_archive_target_probe":
		fields = "status success healthy ok endpoint_id latency_ms duration_ms error_code model decision categories http_status retryable checked_at token_applied"
	case "newapi_conversation_get":
		fields = "id request_id user_id username group_id group_code group_name model protocol message_count byte_size content created_at expires_at"
	case "newapi_conversations_list":
		fields = "id request_id user_id username group_id group_code group_name model protocol message_count byte_size created_at expires_at"
	case "newapi_security_events", "newapi_security_event_get":
		fields = "id request_id user_id username group_id group_code group_name channel_id channel_name model provider protocol source endpoint stage decision action risk_level risk_score safety categories matched_scanners unknown_categories matched_keywords created_at expires_at prompt_hash prompt_length prompt_truncated prompt_available message_count redacted_preview full_prompt context_segments role kind start end text api_key_id api_key_name channel_groups token_groups token_group_mode code name config_version guard_endpoint_id latency_ms chunk_total error_code user_policy_count policy_window_hours user_cyber_policy_count cyber_policy_window_hours"
	case "newapi_model_guard_records":
		fields = "id request_id channel_id channel_name group_id group group_name requested_model expected_upstream_models actual_upstream_model detection_source config_version reason created_at consecutive_mismatches failure_threshold channel_disabled"
	}
	if fields == "" {
		return nil, false, nil
	}
	projected, err := mcpProjectNamedFields(raw, strings.Fields(fields), 0)
	return projected, true, err
}

// mcpProjectNamedFields 是显式业务响应白名单；动态键配置必须使用独立类型处理。
func mcpProjectNamedFields(raw json.RawMessage, fields []string, depth int) (json.RawMessage, error) {
	if depth > 32 {
		return nil, errors.New("response nesting too deep")
	}
	switch common.GetJsonType(raw) {
	case "object":
		var object map[string]json.RawMessage
		if err := common.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		result := map[string]json.RawMessage{}
		for key, value := range object {
			allowed := key == "success" || key == "data" || key == "items" || key == "total" || key == "page" || key == "page_size"
			for _, field := range fields {
				allowed = allowed || key == field
			}
			if !allowed {
				continue
			}
			child, err := mcpProjectNamedFields(value, fields, depth+1)
			if err != nil {
				return nil, err
			}
			result[key] = child
		}
		return common.Marshal(result)
	case "array":
		var array []json.RawMessage
		if err := common.Unmarshal(raw, &array); err != nil {
			return nil, err
		}
		for i, value := range array {
			child, err := mcpProjectNamedFields(value, fields, depth+1)
			if err != nil {
				return nil, err
			}
			array[i] = child
		}
		return common.Marshal(array)
	default:
		return raw, nil
	}
}
