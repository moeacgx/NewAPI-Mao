package service

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// 投影仅保留已知结构字段与元数据。自由文本、配置、原始 payload 和新增未知字段
// 默认丢弃，不能靠枚举 secret/key 来保证未知业务接口不会泄露凭据。
var mcpResponseContainers = strings.Fields("data items list records rows results channels models groups group_details tags users vendors plans subscriptions tasks instances buckets summary stats totals trend points series pagination capabilities resources actions events deliveries activities categories endpoints permissions")
var mcpResponseMetadata = strings.Fields("id user_id channel_id vendor_id group_id group_ids model_id plan_id subscription_id task_id key_id type status enabled active available success total count page p page_size limit offset has_more created_at updated_at expires_at expired_time deleted_at started_at finished_at start_time end_time submit_time finish_time duration duration_ms elapsed_time response_time test_time timestamp time quota used_quota remain_quota unlimited_quota request_count prompt_tokens completion_tokens total_tokens input_tokens output_tokens cached_tokens balance weight priority sort sort_order version progress completed failed pending running total_count success_count failure_count error_count total_requests successful_requests failed_requests average_latency latency latency_ms p50 p95 p99 status_code http_status_code retries retry_count amount money price currency period period_unit period_count reset_period reset_period_seconds total_amount used_amount remaining_amount rate ratio requests tokens rpm tpm healthy online uptime memory cpu cache_hits cache_misses cache_size hit_rate scope name username display_name model models model_name channel_name group_name group_names vendor_name plan_name code group groups category resource action effect built_in superuser grants read write operate sensitive_write secret_view open processing resolved closed")

func ProjectMCPResponse(tool MCPTool, raw []byte, status int) ([]byte, bool, error) {
	if len(raw) > mcpResponseLimit(tool) {
		return nil, false, errors.New("response too large")
	}
	var object map[string]json.RawMessage
	if err := common.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, false, errors.New("business response must be a JSON object")
	}
	success := status >= 200 && status < 300
	if value, exists := object["success"]; exists {
		var ok bool
		if err := common.Unmarshal(value, &ok); err != nil {
			return nil, false, errors.New("invalid business response")
		}
		success = success && ok
	}
	switch tool.Name {
	case "newapi_waffo_pancake_catalog", "newapi_waffo_pancake_pair_create", "newapi_waffo_pancake_save", "newapi_waffo_pancake_subscription_product_create", "newapi_waffo_pancake_subscription_product_options":
		var message string
		if common.Unmarshal(object["message"], &message) != nil || message != "success" {
			success = false
		}
	}
	if tool.Name == "newapi_channel_test" {
		// 测试接口的安全诊断位于顶层；失败也保留计时及固定错误枚举，绝不转发原文。
		result := map[string]any{"success": success}
		if !success {
			result["status"] = status
			result["message"] = "Business request failed"
		}
		var elapsed float64
		if value := object["time"]; common.GetJsonType(value) == "number" && common.Unmarshal(value, &elapsed) == nil && elapsed >= 0 {
			result["time"] = value
		}
		var modelName string
		if value := object["upstream_response_model_name"]; common.GetJsonType(value) == "string" && common.Unmarshal(value, &modelName) == nil && validateMCPModelName(modelName) == nil {
			result["upstream_response_model_name"] = value
		}
		var code string
		if value := object["error_code"]; common.GetJsonType(value) == "string" && common.Unmarshal(value, &code) == nil {
			// 仅宿主的稳定枚举可见，适配器拼接的上游 code 不作为安全元数据。
			for _, allowed := range strings.Fields("invalid_request sensitive_words_detected violation_fee.grok.csam count_token_failed model_price_error invalid_api_type json_marshal_failed do_request_failed get_channel_failed channel_concurrency_limit gen_relay_info_failed channel:no_available_key channel:param_override_invalid channel:header_override_invalid channel:model_mapped_error channel:aws_client_error channel:invalid_key channel:response_time_exceeded read_request_body_failed convert_request_failed access_denied bad_request_body read_response_body_failed bad_response_status_code bad_response bad_response_body empty_response aws_invoke_error model_not_found prompt_blocked query_data_error update_data_error insufficient_user_quota pre_consume_token_quota_failed cyber_policy biological_risk session_blocked_by_security_policy session_blocked_by_cyber_policy prompt_guard_blocked") {
				if code == allowed {
					result["error_code"] = value
					break
				}
			}
		}
		projected, err := common.Marshal(result)
		return projected, success, err
	}
	// 失败正文经常拼接上游响应、SQL 或输入，不向客户端转发。
	if !success {
		result, _ := common.Marshal(map[string]any{"success": false, "status": status, "message": "Business request failed"})
		return result, false, nil
	}
	for _, project := range []func(MCPTool, []byte) ([]byte, bool, error){
		projectMCPFullSystemResponse,
		projectMCPRatioResponse,
		projectMCPFullChannelResponse,
		projectMCPFullAccountResponse,
		projectMCPFullCommerceResponse,
		projectMCPFullAnalyticsResponse,
		projectMCPFullOperationResponse,
	} {
		projected, handled, err := project(tool, raw)
		if handled {
			return projected, success, err
		}
	}
	if tool.Category == "models" {
		var names map[string]string
		var err error
		raw, err = enrichMCPModelGroupNames(raw, &names, 0)
		if err != nil {
			return nil, false, err
		}
	}
	projected, err := projectMCPValue(tool, raw, "", 0)
	if err != nil {
		return nil, false, err
	}
	if len(projected) > 48*1024 {
		return nil, false, errors.New("projected response too large")
	}
	return projected, success, nil
}

func projectMCPValue(tool MCPTool, raw json.RawMessage, field string, depth int) (json.RawMessage, error) {
	if depth > 32 {
		return nil, errors.New("response nesting too deep")
	}
	switch common.GetJsonType(raw) {
	case "object":
		var object map[string]json.RawMessage
		if err := common.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		out := make(map[string]json.RawMessage)
		for key, value := range object {
			lower := strings.ToLower(key)
			modelRead := tool.Name == "newapi_model_get" || tool.Name == "newapi_models_list" || tool.Name == "newapi_models_search"
			if modelRead && (lower == "bound_channels" || lower == "matched_models" || lower == "matched_count" || lower == "quota_types") || tool.Name == "newapi_models_missing" && depth == 0 && key == "data" {
				child, err := projectMCPModelAssociation(lower, value)
				if err != nil {
					return nil, err
				}
				out[key] = child
				continue
			}
			statistic := mcpStatisticsField(tool.Name, lower)
			valueType := common.GetJsonType(value)
			// 统计 token 字段仅对数值或缺失样本的 null 例外，不能借白名单携带凭据文本。
			numericStatistic := statistic && (valueType == "number" || valueType == "null")
			if mcpEditableMetadataField(tool.Name, lower) {
				if err := validateMCPEditableMetadata(lower, value); err != nil {
					return nil, err
				}
				out[key] = value
				continue
			}
			if strings.Contains(lower, "key") && lower != "key_id" || strings.Contains(lower, "token") && !numericStatistic && lower != "prompt_tokens" && lower != "completion_tokens" && lower != "total_tokens" && lower != "input_tokens" && lower != "output_tokens" && lower != "cached_tokens" && lower != "tokens" || strings.Contains(lower, "secret") || strings.Contains(lower, "password") {
				continue
			}
			// 优惠/兑换 code 也是凭据，不能当作普通业务 code 返回。
			if tool.Category == "marketing" && lower == "code" {
				continue
			}
			allowed := statistic
			for _, name := range mcpResponseContainers {
				allowed = allowed || lower == name
			}
			for _, name := range mcpResponseMetadata {
				allowed = allowed || lower == name
			}
			if tool.Category == "models" && (lower == "enable_groups" || lower == "enable_group_names") {
				allowed = true
			}
			if !allowed {
				continue
			}
			child, err := projectMCPValue(tool, value, lower, depth+1)
			if err != nil {
				return nil, err
			}
			out[key] = child
		}
		return common.Marshal(out)
	case "array":
		var array []json.RawMessage
		if err := common.Unmarshal(raw, &array); err != nil {
			return nil, err
		}
		out := make([]json.RawMessage, 0, len(array))
		for _, item := range array {
			child, err := projectMCPValue(tool, item, field, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, child)
		}
		return common.Marshal(out)
	case "string":
		// 模型 ID 中的 SparkDesk 等合法片段不能按普通文本中的 sk- 凭据标记误删。
		if tool.Name == "newapi_channel_models" && field == "id" || tool.Name == "newapi_channel_models_enabled" && field == "data" && depth == 2 {
			var name string
			if common.Unmarshal(raw, &name) != nil || validateMCPModelName(name) != nil {
				return nil, errors.New("unsafe model catalog name")
			}
			return raw, nil
		}
		// data 等容器若意外携带原始字符串，一律隐藏；只有明确的元数据标量可见。
		allowed := mcpStatisticsField(tool.Name, field)
		for _, name := range mcpResponseMetadata {
			allowed = allowed || name == field
		}
		if tool.Category == "models" && (field == "enable_groups" || field == "enable_group_names") {
			allowed = true
		}
		if field == "data" && depth == 2 && (tool.Name == "newapi_channel_models" || tool.Name == "newapi_channel_models_enabled") {
			allowed = true
		}
		if !allowed {
			return json.RawMessage(`null`), nil
		}
		var text string
		if err := common.Unmarshal(raw, &text); err != nil {
			return nil, err
		}
		if len(text) > 256 || strings.ContainsAny(text, "\r\n") || strings.Contains(text, "://") || strings.Contains(text, "Bearer ") || strings.Contains(text, "mcp_") || strings.Contains(text, "sk-") || strings.HasPrefix(strings.TrimSpace(text), "{") || strings.HasPrefix(strings.TrimSpace(text), "[") {
			return json.RawMessage(`"[redacted]"`), nil
		}
		return raw, nil
	default:
		return raw, nil
	}
}
