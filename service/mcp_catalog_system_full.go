package service

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

func mcpFullSystemTools() []MCPTool {
	tools := []MCPTool{
		{Name: "newapi_health_check", Path: "/api/status/test", Description: "检查数据库连接和 HTTP 服务统计。"},
		{Name: "newapi_update_release", Path: "/api/status/github-latest-release", Description: "查询宿主配置的发行源及当前版本更新信息，会访问外部发行服务。"},
		{Name: "newapi_self_update", Method: http.MethodPost, Path: "/api/status/self-update", Write: true, Description: "按指定 tag_name 执行宿主更新，可能重启当前进程。沿用固定发行源与校验；连接中断时先查询版本，不能更换 request_id 盲目重试。"},
		{Name: "newapi_settings_list", Path: "/api/option/", Description: "分页读取系统配置。query.key 精确选项名、keyword 筛选、p/page_size 分页。敏感值隐藏，返回 redacted 标记；JSON 配置仍为字符串。"},
		{Name: "newapi_setting_update", Method: http.MethodPut, Path: "/api/option/", Write: true, Description: "更新一个系统选项，包括模型定价、站点、配额、支付和集成配置。先读取原值，value 一律字符串，JSON 配置使用序列化字符串。凭据仅作为输入。保留宿主校验与权限门禁。"},
		{Name: "newapi_channel_affinity_cache", Path: "/api/option/channel_affinity_cache", Description: "读取渠道亲和缓存统计。"},
		{Name: "newapi_channel_affinity_usage", Path: "/api/log/channel_affinity_usage_cache", Description: "读取亲和规则用量缓存，query.rule_name 和 key_fp 必填，using_group 可选。key_fp 是指纹，不是凭据。"},
		{Name: "newapi_channel_affinity_clear", Method: http.MethodDelete, Path: "/api/option/channel_affinity_cache", Write: true, Description: "清理指定 rule_name 的亲和缓存，或显式 all=true 清空全部。"},
		{Name: "newapi_ratio_reset", Method: http.MethodPost, Path: "/api/option/rest_model_ratio", Write: true, Description: "恢复宿主模型默认倍率配置，将覆盖当前倍率。"},
		{Name: "newapi_ratio_sources", Path: "/api/ratio_sync/channels", Description: "查询价格同步来源和内置预设。"},
		{Name: "newapi_ratio_fetch", Method: http.MethodPost, Path: "/api/ratio_sync/fetch", Write: true, Description: "请求价格来源并比较差异，不自动应用价格。可能访问外部来源；timeout 以秒计，最多 20。"},
		{Name: "newapi_performance_logs", Path: "/api/performance/logs", Description: "读取服务日志文件元数据，不读取日志正文。"},
		{Name: "newapi_performance_logs_cleanup", Method: http.MethodDelete, Path: "/api/performance/logs", Write: true, Description: "清理服务日志文件。query.mode=by_days 按保留天数、by_count 按保留文件数；query.value 为正整数。"},
		{Name: "newapi_disk_cache_cleanup", Method: http.MethodDelete, Path: "/api/performance/disk_cache", Write: true, Description: "清理宿主判定为不活跃的磁盘缓存。"},
		{Name: "newapi_performance_reset", Method: http.MethodPost, Path: "/api/performance/reset_stats", Write: true, Description: "重置性能统计计数，不改变计费数据。"},
		{Name: "newapi_performance_gc", Method: http.MethodPost, Path: "/api/performance/gc", Write: true, Description: "请求宿主执行垃圾回收。"},
		{Name: "newapi_log_cleanup_start", Method: http.MethodPost, Path: "/api/system-task/log-cleanup", Write: true, Description: "创建异步日志清理任务，query.target_timestamp 为清理截止 Unix 秒。返回任务 ID 后通过 system_task 工具查询状态。"},
		{Name: "newapi_instances_cleanup", Method: http.MethodDelete, Path: "/api/system-info/stale-instances", Write: true, Description: "删除已过期的实例心跳记录，不停止运行实例。"},
		{Name: "newapi_instance_delete", Method: http.MethodDelete, Path: "/api/system-info/instances/:node_name", Write: true, Description: "按节点名称删除过期实例记录；在线节点由宿主拒绝。"},
		{Name: "newapi_security_config", Path: "/api/security-audit/config", Description: "读取安全审计完整可编辑配置和 config_version，节点令牌仅返回配置状态。"},
		{Name: "newapi_security_config_update", Method: http.MethodPut, Path: "/api/security-audit/config", Write: true, Description: "保存安全审计配置。先读取并提交 expected_version；完整保存字段必填，节点 token_action 使用 keep/replace/clear。"},
		{Name: "newapi_security_policy", Path: "/api/security-audit/builtin-policy", Description: "读取内置安全策略配置与版本。"},
		{Name: "newapi_security_policy_update", Method: http.MethodPut, Path: "/api/security-audit/builtin-policy", Write: true, Description: "按 expected_version 更新内置安全策略，保留宿主版本冲突检查。"},
		{Name: "newapi_security_policy_channels", Path: "/api/security-audit/builtin-policy/channels", Description: "查询安全策略可选渠道。"},
		{Name: "newapi_security_policy_tags", Path: "/api/security-audit/builtin-policy/channel-tags", Description: "查询安全策略可选渠道标签。"},
		{Name: "newapi_security_policy_groups", Path: "/api/security-audit/builtin-policy/groups", Description: "查询安全策略可选业务分组，name 展示、code 提交。"},
		{Name: "newapi_security_probe", Method: http.MethodPost, Path: "/api/security-audit/endpoints/probe", Write: true, Description: "主动探测 Guard 节点，可能产生上游调用。凭据仅输入，结果不回显原始响应。"},
		{Name: "newapi_security_events", Path: "/api/security-audit/events", Description: "分页筛选安全审计事件，不返回完整提示词。"},
		{Name: "newapi_security_event_get", Path: "/api/security-audit/events/:id", Description: "读取安全审计事件诊断和上下文，属于独立授权的敏感内容读取。"},
		{Name: "newapi_security_event_delete", Method: http.MethodDelete, Path: "/api/security-audit/events/:id", Write: true, Description: "删除一个安全审计事件及关联任务。"},
		{Name: "newapi_security_events_delete", Method: http.MethodPost, Path: "/api/security-audit/events/batch-delete", Write: true, Description: "按明确 ID 列表批量删除安全审计事件。"},
		{Name: "newapi_security_delete_preview", Method: http.MethodPost, Path: "/api/security-audit/events/delete-preview", Description: "预览按筛选删除的影响，返回当前操作者绑定的短期 confirmation_token；不删除数据。"},
		{Name: "newapi_security_delete_filtered", Method: http.MethodPost, Path: "/api/security-audit/events/delete-by-filter", Write: true, Description: "提交预览时相同 filter、confirmation_token 和 confirm=true 执行删除；不可跳过预览。"},
		{Name: "newapi_archive_config", Path: "/api/security-audit/request-archive/config", Description: "读取请求归档配置、版本和存储目标，凭据只返回已配置状态。"},
		{Name: "newapi_archive_config_update", Method: http.MethodPut, Path: "/api/security-audit/request-archive/config", Write: true, Description: "完整保存请求归档配置，expected_version 防并发覆盖，凭据通过 keep/replace/clear 管理。"},
		{Name: "newapi_archive_target_probe", Method: http.MethodPost, Path: "/api/security-audit/request-archive/targets/probe", Write: true, Description: "主动检测归档存储目标，沿用宿主路径和凭据校验。"},
		{Name: "newapi_conversation_config", Path: "/api/extensions/conversation-archive/config", Description: "读取对话归档配置与版本。"},
		{Name: "newapi_conversation_config_update", Method: http.MethodPut, Path: "/api/extensions/conversation-archive/config", Write: true, Description: "完整保存对话归档配置，expected_version 防止覆盖他人修改。"},
		{Name: "newapi_conversations_list", Path: "/api/extensions/conversation-archive/conversations", Description: "分页查询对话归档，支持 group_codes、user_ids 逗号分隔筛选和起止时间。"},
		{Name: "newapi_conversation_get", Path: "/api/extensions/conversation-archive/conversations/:id", Description: "读取指定归档正文；仅授予需要诊断用户对话的管理员。"},
		{Name: "newapi_conversations_clear", Method: http.MethodPost, Path: "/api/extensions/conversation-archive/conversations/clear", Write: true, Description: "清空对话归档，必须显式 confirm=true，不可撤销。"},
		{Name: "newapi_model_guard_config", Path: "/api/extensions/upstream-model-guard/config", Description: "读取上游模型守卫配置与版本。"},
		{Name: "newapi_model_guard_update", Method: http.MethodPut, Path: "/api/extensions/upstream-model-guard/config", Write: true, Description: "按 expected_version 保存上游模型守卫配置。"},
		{Name: "newapi_model_guard_channels", Path: "/api/extensions/upstream-model-guard/channels", Description: "分页查询可排除的渠道。"},
		{Name: "newapi_model_guard_records", Path: "/api/extensions/upstream-model-guard/records", Description: "分页查询模型守卫检测记录。"},
	}
	filter := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{}}
	fields := filter["properties"].(map[string]any)
	for _, key := range []string{"source", "stage", "decision", "action", "risk_level", "endpoint", "request_id", "prompt_hash", "keyword", "username"} {
		fields[key] = map[string]any{"type": "string", "maxLength": 256}
	}
	for _, key := range []string{"user_id", "token_id", "group_id", "channel_id"} {
		fields[key] = map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483647}
	}
	for _, key := range []string{"start_at", "end_at"} {
		fields[key] = map[string]any{"type": "integer", "minimum": 0, "maximum": int64(253402300799)}
	}
	for i := range tools {
		tool := &tools[i]
		tool.Category = "system"
		tool.MinRole = common.RoleRootUser
		if tool.Method == "" {
			tool.Method = http.MethodGet
		}
		tool.InputSchema = mcpToolInputSchema(*tool)
		properties := tool.InputSchema["properties"].(map[string]any)
		if tool.Name == "newapi_instance_delete" {
			properties["path"].(map[string]any)["properties"].(map[string]any)["node_name"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "pattern": "^[A-Za-z0-9][A-Za-z0-9._-]*$"}
		}
		body := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{}}
		switch tool.Name {
		case "newapi_self_update":
			body = mcpTypedBodySchema(struct {
				TagName string `json:"tag_name"`
			}{})
			body["properties"].(map[string]any)["tag_name"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "pattern": "^[A-Za-z0-9_.-]+$"}
		case "newapi_setting_update":
			body = mcpTypedBodySchema(struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}{})
			body["properties"].(map[string]any)["key"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 256, "pattern": "^[A-Za-z][A-Za-z0-9_.-]*$"}
			body["properties"].(map[string]any)["value"] = map[string]any{"type": "string", "maxLength": 524288}
		case "newapi_ratio_fetch":
			body = mcpTypedBodySchema(dto.UpstreamRequest{})
			body["required"] = []string{}
			body["properties"].(map[string]any)["timeout"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 20}
			upstream := body["properties"].(map[string]any)["upstreams"].(map[string]any)["items"].(map[string]any)
			upstream["properties"].(map[string]any)["id"].(map[string]any)["minimum"] = -101
		case "newapi_security_config_update":
			body = mcpTypedBodySchema(PromptAuditUpdateRequest{})
		case "newapi_security_policy_update":
			body = mcpTypedBodySchema(SecurityAuditBuiltinPolicyUpdateRequest{})
		case "newapi_archive_config_update":
			body = mcpTypedBodySchema(RequestArchiveUpdateRequest{})
		case "newapi_archive_target_probe":
			body = mcpTypedBodySchema(RequestArchiveUpdateTarget{})
		case "newapi_conversation_config_update":
			body = mcpTypedBodySchema(ConversationArchiveConfigUpdate{})
		case "newapi_model_guard_update":
			body = mcpTypedBodySchema(UpstreamModelGuardConfigUpdate{})
		case "newapi_security_probe":
			body = mcpTypedBodySchema(struct {
				EndpointID  string `json:"endpoint_id,omitempty"`
				Name        string `json:"name,omitempty"`
				BaseURL     string `json:"base_url,omitempty"`
				Model       string `json:"model,omitempty"`
				TokenAction string `json:"token_action,omitempty"`
				Token       string `json:"token,omitempty"`
				TimeoutMS   *int   `json:"timeout_ms,omitempty"`
				InputLimit  *int   `json:"input_limit,omitempty"`
			}{})
		case "newapi_security_events_delete":
			body = mcpTypedBodySchema(struct {
				IDs []int `json:"ids"`
			}{})
			body["properties"].(map[string]any)["ids"].(map[string]any)["minItems"] = 1
		case "newapi_security_delete_preview":
			body = filter
			tool.InputSchema["required"] = []string{"body"}
		case "newapi_security_delete_filtered":
			body = map[string]any{"type": "object", "additionalProperties": false, "required": []string{"filter", "confirmation_token", "confirm"}, "properties": map[string]any{
				"filter":             filter,
				"confirmation_token": map[string]any{"type": "string", "minLength": 1, "maxLength": 4096},
				"confirm":            map[string]any{"type": "boolean", "enum": []bool{true}},
			}}
		case "newapi_conversations_clear":
			body = map[string]any{"type": "object", "additionalProperties": false, "required": []string{"confirm"}, "properties": map[string]any{"confirm": map[string]any{"type": "boolean", "enum": []bool{true}}}}
		}
		if tool.Method == http.MethodPost || tool.Method == http.MethodPut {
			properties["body"] = body
		}
		queryFields := map[string]any{}
		queryRequired := []string{}
		switch tool.Name {
		case "newapi_channel_affinity_usage":
			queryFields["rule_name"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128}
			queryFields["key_fp"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128}
			queryFields["using_group"] = map[string]any{"type": "string", "maxLength": 128}
			queryRequired = []string{"rule_name", "key_fp"}
		case "newapi_settings_list":
			queryFields["key"] = map[string]any{"type": "string", "maxLength": 256}
			queryFields["keyword"] = map[string]any{"type": "string", "maxLength": 128}
			queryFields["p"] = map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,3}$"}
			queryFields["page_size"] = map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,2}$"}
		case "newapi_performance_logs_cleanup":
			queryFields["mode"] = map[string]any{"type": "string", "enum": []string{"by_count", "by_days"}}
			queryFields["value"] = map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,3}$"}
			queryRequired = []string{"mode", "value"}
		case "newapi_channel_affinity_clear":
			queryFields["all"] = map[string]any{"type": "string", "enum": []string{"true"}}
			queryFields["rule_name"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128}
		case "newapi_log_cleanup_start":
			queryFields["target_timestamp"] = map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,11}$"}
			queryRequired = []string{"target_timestamp"}
		}
		if len(queryFields) > 0 {
			properties["query"] = map[string]any{"type": "object", "properties": queryFields, "required": queryRequired, "additionalProperties": false}
			if len(queryRequired) > 0 {
				tool.InputSchema["required"] = append(tool.InputSchema["required"].([]string), "query")
			}
		}
	}
	return tools
}
