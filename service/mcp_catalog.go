package service

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/authz"
)

// MCPTool 同时驱动协议发现和管理界面，业务路由只允许来自此显式目录。
type MCPTool struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Category    string           `json:"category"`
	Method      string           `json:"-"`
	Path        string           `json:"-"`
	MinRole     int              `json:"min_role"`
	Write       bool             `json:"write"`
	Permission  authz.Permission `json:"-"`
	InputSchema map[string]any   `json:"inputSchema"`
	Annotations map[string]bool  `json:"annotations"`
	// 新目录的 ID 约束由完整 schema 描述，支持宿主定义的字符串 ID 与零值哨兵。
	TypedArguments  bool     `json:"-"`
	StringPathIDs   []string `json:"-"`
	StringQueryIDs  []string `json:"-"`
	TransientResult bool     `json:"-"`
	UploadField     string   `json:"-"`
}

// MCPTools 返回独立目录快照，避免请求之间共享可变的参数 schema。
func MCPTools() []MCPTool {
	tools := []MCPTool{
		{Name: "newapi_context", Category: "system", Description: "连接后首先调用：核对当前身份、角色、授权工具与服务范围，不触发账户结算。"},
		{Name: "newapi_status", Category: "system", Path: "/api/status", Description: "查询站点状态、版本及公开能力。"},
		{Name: "newapi_permissions", Category: "system", Path: "/api/authz/catalog", Description: "查询后台权限定义；此目录不是当前身份的有效权限。"},
		{Name: "newapi_channels_list", Category: "channels", Path: "/api/channel/", Description: "分页查询渠道摘要。query 支持 p、page_size、type、tag；不返回密钥、上游地址和覆盖配置。"},
		{Name: "newapi_channels_search", Category: "channels", Path: "/api/channel/search", Description: "搜索渠道。query 支持 keyword、group、model、p、page_size；group 使用业务 code。"},
		{Name: "newapi_channel_get", Category: "channels", Path: "/api/channel/:id", Description: "按渠道 ID 获取脱敏详情。"},
		{Name: "newapi_channel_models", Category: "channels", Path: "/api/channel/models", Description: "分页查询宿主内置模型参考目录，data 含 items、total、page、page_size；query 支持 p、page_size，默认每页 20。不是指定渠道的实时上游模型列表。"},
		{Name: "newapi_channel_models_enabled", Category: "channels", Path: "/api/channel/models_enabled", Description: "查询已启用渠道中配置的模型；不保证当前用户已获分组授权、模型已定价或上游实时可用。"},
		{Name: "newapi_channel_operations", Category: "channels", Path: "/api/channel/ops", Description: "查询渠道运行概况。"},
		{Name: "newapi_channel_priority", Category: "channels", Path: "/api/channel/priority/order", Description: "查询渠道优先级排序。"},
		{Name: "newapi_channel_status_set", Category: "channels", Method: http.MethodPost, Path: "/api/channel/:id/status", Write: true, Permission: authz.ChannelOperate, Description: "启用或禁用单个渠道。body.status 为 1（启用）或 2（手动禁用）。"},
		{Name: "newapi_channel_test", Category: "channels", Path: "/api/channel/test/:id", Write: true, Permission: authz.ChannelOperate, Description: "主动测试一个渠道并更新测试结果，可能产生上游费用。query 可提供 model、endpoint_type、stream；支持的测试协议见 schema，不包含任意请求体或媒体任务测试；必须提供 request_id。"},
		{Name: "newapi_channel_balance_refresh", Category: "channels", Path: "/api/channel/update_balance/:id", Write: true, Permission: authz.ChannelOperate, Description: "请求上游刷新单个渠道余额，属于主动操作。"},
		{Name: "newapi_channel_create", Category: "channels", Method: http.MethodPost, Path: "/api/channel/", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "创建渠道，须单独授予敏感写权限。body 使用后台 AddChannel 契约（mode、channel）；密钥只用作输入，不进入审计。"},
		{Name: "newapi_channel_update", Category: "channels", Method: http.MethodPut, Path: "/api/channel/", Write: true, Permission: authz.ChannelWrite, Description: "更新渠道路由。body 只允许 id、name、models、group、priority、weight；models 是逗号分隔的完整覆盖列表，增删前先读取合并，不是追加补丁。状态使用单独工具，密钥和上游地址不能通过此工具修改。"},
		{Name: "newapi_channel_delete", Category: "channels", Method: http.MethodDelete, Path: "/api/channel/:id", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "删除指定渠道，此操作不可通过 MCP 撤销。"},
		{Name: "newapi_channel_analytics_summary", Category: "channels", Path: "/api/channel-analytics/summary", Description: "查询渠道分析汇总。query 使用 start_timestamp、end_timestamp。"},
		{Name: "newapi_channel_analytics_trend", Category: "channels", Path: "/api/channel-analytics/trend", Description: "查询渠道请求趋势。query 使用 start_timestamp、end_timestamp。"},
		{Name: "newapi_channel_analytics_channels", Category: "channels", Path: "/api/channel-analytics/channels", Description: "分页查询各渠道请求和错误指标。"},
		{Name: "newapi_channel_analytics_models", Category: "channels", Path: "/api/channel-analytics/channels/:id/models", Description: "查询指定渠道的模型指标。"},
		{Name: "newapi_channel_stability", Category: "channels", Path: "/api/channel-analytics/stability", Description: "查询渠道稳定性。"},
		{Name: "newapi_channel_status_codes", Category: "channels", Path: "/api/channel-analytics/status-codes", Description: "查询渠道响应状态码分布。"},
		{Name: "newapi_channel_failures", Category: "channels", Path: "/api/channel-analytics/failures", Description: "查询渠道失败统计；不返回原始日志正文。"},
		{Name: "newapi_users_list", Category: "users", Path: "/api/user/", Description: "分页查询用户摘要，隐藏认证和个人联系信息。"},
		{Name: "newapi_users_search", Category: "users", Path: "/api/user/search", Description: "按 query.keyword 搜索用户。"},
		{Name: "newapi_user_get", Category: "users", Path: "/api/user/:id", Description: "查询指定用户摘要，仍遵守目标角色限制。"},
		{Name: "newapi_groups_list", Category: "groups", Path: "/api/group/details", Description: "查询分组详情，name 为显示名称，id/code 用于业务关联，不要把显示名称作为提交标识。"},
		{Name: "newapi_group_tags_list", Category: "groups", Path: "/api/group/tags", Description: "查询分组标签与分组 ID 绑定。"},
		{Name: "newapi_group_tag_create", Category: "groups", Method: http.MethodPost, Path: "/api/group/tags", Write: true, Description: "创建展示标签。body 包含 name，可选 description、icons、sort_order、group_ids、match_keywords；不改变分组计费或授权。"},
		{Name: "newapi_group_tag_update", Category: "groups", Method: http.MethodPut, Path: "/api/group/tags/:id", Write: true, Description: "保存标签完整配置，先读取现有标签再修改；缺省集合会清空原绑定。"},
		{Name: "newapi_group_tag_delete", Category: "groups", Method: http.MethodDelete, Path: "/api/group/tags/:id", Write: true, Description: "删除指定展示标签，不删除分组。"},
		{Name: "newapi_prefill_groups", Category: "groups", Path: "/api/prefill_group/", Description: "查询预设分组配置。"},
		{Name: "newapi_models_list", Category: "models", Path: "/api/models/", Description: "分页查询模型管理元数据，区别于用户可用模型接口。"},
		{Name: "newapi_models_search", Category: "models", Path: "/api/models/search", Description: "按 query.keyword 搜索模型元数据。"},
		{Name: "newapi_model_get", Category: "models", Path: "/api/models/:id", Description: "按元数据 ID 查询模型。"},
		{Name: "newapi_models_missing", Category: "models", Path: "/api/models/missing", Description: "查询尚未建立元数据的模型。"},
		{Name: "newapi_model_create", Category: "models", Method: http.MethodPost, Path: "/api/models/", Write: true, Description: "创建模型展示元数据，不自动添加渠道或设置价格。body.model_name 必填；status、sync_official 默认 0，启用请显式传 status=1；vendor_id=0 表示不绑定供应商；endpoints 为 JSON 字符串。"},
		{Name: "newapi_model_update", Category: "models", Method: http.MethodPut, Path: "/api/models/", Write: true, Description: "完整保存模型元数据。先获取详情，提交 schema 中全部编辑字段；省略的空字符串、vendor_id=0 等需补齐，避免清空原配置。不修改渠道或价格。"},
		{Name: "newapi_model_delete", Category: "models", Method: http.MethodDelete, Path: "/api/models/:id", Write: true, Description: "删除指定模型元数据。"},
		{Name: "newapi_vendors_list", Category: "models", Path: "/api/vendors/", Description: "分页查询模型供应商。"},
		{Name: "newapi_vendor_get", Category: "models", Path: "/api/vendors/:id", Description: "查询供应商详情。"},
		{Name: "newapi_vendor_create", Category: "models", Method: http.MethodPost, Path: "/api/vendors/", Write: true, Description: "创建供应商元数据。body 包含 name，可选 description、icon、status。"},
		{Name: "newapi_vendor_update", Category: "models", Method: http.MethodPut, Path: "/api/vendors/", Write: true, Description: "更新供应商元数据。body 包含 id 和完整供应商配置。"},
		{Name: "newapi_vendor_delete", Category: "models", Method: http.MethodDelete, Path: "/api/vendors/:id", Write: true, Description: "删除指定供应商元数据。"},
		{Name: "newapi_logs_list", Category: "usage", Path: "/api/log/", Description: "分页查询用量日志，query 使用 start_timestamp、end_timestamp、type、model_name、username；原始 content/other 不返回。"},
		{Name: "newapi_usage_stats", Category: "usage", Path: "/api/log/stat", Description: "查询用量统计，query 使用 start_timestamp、end_timestamp；向用户展示消费使用 money_display 中的站内换算金额，原始 quota 仅供精确计算。"},
		{Name: "newapi_active_requests", Category: "usage", Path: "/api/log/active", Description: "查询一个用户的活动请求，query 必须提供 user_id 或精确 username，分页用 page/page_size；保留 instance/cluster 范围。"},
		{Name: "newapi_quota_history", Category: "usage", Path: "/api/data/", Description: "查询额度历史，query 使用 start_timestamp、end_timestamp。"},
		{Name: "newapi_quota_by_user", Category: "usage", Path: "/api/data/users", Description: "按用户查询额度统计。query 使用 start_timestamp、end_timestamp。"},
		{Name: "newapi_flow_history", Category: "usage", Path: "/api/data/flow", Description: "查询额度流量历史。query 使用 start_timestamp、end_timestamp。"},
		{Name: "newapi_revenue", Category: "usage", Path: "/api/data/revenue", Description: "查询收入统计。query 使用 start_timestamp、end_timestamp；金额口径以接口字段为准。"},
		{Name: "newapi_tasks_list", Category: "tasks", Path: "/api/task/", Description: "分页查询异步任务状态，排除原始请求、媒体和上游响应内容。"},
		{Name: "newapi_midjourney_tasks", Category: "tasks", Path: "/api/mj/", Description: "分页查询 Midjourney 任务摘要。"},
		{Name: "newapi_my_tokens_list", Category: "tokens", Path: "/api/token/", Description: "分页查询当前操作者本人的推理令牌元数据，不返回 key，不支持跨用户。"},
		{Name: "newapi_my_tokens_search", Category: "tokens", Path: "/api/token/search", Description: "搜索本人推理令牌，query 支持 keyword。"},
		{Name: "newapi_my_token_get", Category: "tokens", Path: "/api/token/:id", Description: "查询本人指定令牌的脱敏元数据。"},
		{Name: "newapi_my_token_delete", Category: "tokens", Method: http.MethodDelete, Path: "/api/token/:id", Write: true, Description: "删除本人指定推理令牌，删除后使用该令牌的客户端将无法调用。"},
		{Name: "newapi_tickets_list", Category: "tickets", Path: "/api/ticket-admin", Description: "分页查询管理工单。query 支持 status、keyword、p、page_size；不返回正文。"},
		{Name: "newapi_ticket_get", Category: "tickets", Path: "/api/ticket-admin/:id", Description: "查询工单及消息元数据，排除用户正文。"},
		{Name: "newapi_ticket_stats", Category: "tickets", Path: "/api/ticket-admin/stats", Description: "查询待处理、进行中、已解决、已关闭工单统计。"},
		{Name: "newapi_ticket_status_set", Category: "tickets", Method: http.MethodPut, Path: "/api/ticket-admin/:id/status", Write: true, Permission: authz.TicketAdminWrite, Description: "变更工单状态，body.status 为 open、processing、resolved 或 closed。"},
		{Name: "newapi_subscription_plans", Category: "billing", Path: "/api/subscription/admin/plans", Description: "查询订阅计划，不绑定、不重置和不扣费。"},
		{Name: "newapi_user_subscriptions", Category: "billing", Path: "/api/subscription/admin/users/:id/subscriptions", Description: "查询指定用户的订阅权益和有效期。"},
		{Name: "newapi_topups_list", Category: "billing", Path: "/api/user/topup", Description: "分页查询充值订单元数据，不执行补单或付款。"},
		{Name: "newapi_invoices_list", Category: "billing", Path: "/api/user/invoice", Description: "分页查询发票记录，个人身份、支付信息和发票文件不返回。"},
		{Name: "newapi_redemptions_list", Category: "marketing", Path: "/api/redemption/", Description: "分页查询兑换码状态与额度，兑换密钥不返回。"},
		{Name: "newapi_redemption_get", Category: "marketing", Path: "/api/redemption/:id", Description: "查询指定兑换码元数据，不返回兑换凭据。"},
		{Name: "newapi_promo_codes_list", Category: "marketing", Path: "/api/promo_code/", Description: "分页查询优惠码管理记录，凭据字段脱敏。"},
		{Name: "newapi_benefit_activities", Category: "marketing", Path: "/api/benefit/admin/activities", Description: "查询福利活动清单。"},
		{Name: "newapi_benefit_report", Category: "marketing", Path: "/api/benefit/admin/activities/:id/report", Write: true, Description: "查询福利统计；宿主会同步处理过期券，因此必须 request_id，重复标识不再次处理。"},
		{Name: "newapi_affiliate_records", Category: "affiliate", Path: "/api/affiliate/admin/records", Write: true, Description: "查询返佣摘要；宿主会结算成熟返佣，因此必须 request_id，重复标识不再次结算。"},
		{Name: "newapi_affiliate_invitations", Category: "affiliate", Path: "/api/affiliate/admin/invitations", Description: "查询邀请关系元数据。"},
		{Name: "newapi_affiliate_withdrawals", Category: "affiliate", Path: "/api/affiliate/admin/withdrawals", Description: "查询提现状态，排除收款账户和付款凭据。"},
		{Name: "newapi_prompts_list", Category: "prompts", Path: "/api/prompt-center/admin/", Description: "查询提示词与技能元数据，排除正文、源码和仓库凭据。"},
		{Name: "newapi_prompt_categories", Category: "prompts", Path: "/api/prompt-center/categories/", Description: "查询提示词分类。"},
		{Name: "newapi_plugins_list", Category: "extensions", Path: "/api/plugin/task", Description: "查询任务插件清单与版本，排除源码和配置。"},
		{Name: "newapi_plugin_runtime", Category: "extensions", Path: "/api/plugin/task/runtime/status", Description: "查询任务插件运行时状态，不启停运行时。"},
		{Name: "newapi_extensions_list", Category: "extensions", Path: "/api/extension-admin/", MinRole: common.RoleRootUser, Description: "查询已安装扩展清单，不安装、刷新或执行扩展。"},
		{Name: "newapi_notification_events", Category: "notifications", Path: "/api/notification/event-types", MinRole: common.RoleRootUser, Description: "查询可配置的通知事件类型。"},
		{Name: "newapi_notification_tasks", Category: "notifications", Path: "/api/notification/tasks", MinRole: common.RoleRootUser, Description: "查询通知任务元数据，排除目标和模板内容，不投递消息。"},
		{Name: "newapi_notification_deliveries", Category: "notifications", Path: "/api/notification/deliveries", MinRole: common.RoleRootUser, Description: "分页查询通知投递状态，排除消息正文和目标地址。"},
		{Name: "newapi_system_tasks", Category: "system", Path: "/api/system-task/list", MinRole: common.RoleRootUser, Description: "查询系统任务清单。"},
		{Name: "newapi_system_task_current", Category: "system", Path: "/api/system-task/current", MinRole: common.RoleRootUser, Description: "查询当前系统任务。"},
		{Name: "newapi_system_task_get", Category: "system", Path: "/api/system-task/:task_id", MinRole: common.RoleRootUser, Description: "按 task_id 查询系统任务状态。"},
		{Name: "newapi_system_instances", Category: "system", Path: "/api/system-info/instances", MinRole: common.RoleRootUser, Description: "查询实例状态和心跳，不返回内部网络地址。"},
		{Name: "newapi_performance", Category: "system", Path: "/api/performance/stats", MinRole: common.RoleRootUser, Description: "查询性能和缓存统计，不清理缓存或执行 GC。"},
		{Name: "newapi_security_runtime", Category: "security", Path: "/api/security-audit/runtime", MinRole: common.RoleRootUser, Description: "查询安全审计运行状态，不返回审计正文。"},
		{Name: "newapi_archive_runtime", Category: "security", Path: "/api/security-audit/request-archive/runtime", MinRole: common.RoleRootUser, Description: "查询请求归档运行状态，不返回归档内容。"},
		{Name: "newapi_deployment_settings", Category: "deployments", Path: "/api/deployments/settings", Description: "查询模型部署能力是否配置，不读取供应商凭据。"},
	}
	extra := append(mcpFullSystemTools(), mcpFullChannelTools()...)
	extra = append(extra, mcpFullAccountTools()...)
	extra = append(extra, mcpFullCommerceTools()...)
	extra = append(extra, mcpFullAnalyticsTools()...)
	extra = append(extra, mcpFullOperationTools()...)
	for i := range extra {
		extra[i].TypedArguments = true
	}
	tools = append(tools, extra...)
	for i := range tools {
		tool := &tools[i]
		if tool.Category == "users" || tool.Category == "tokens" {
			tool.Description += " 余额和消费展示优先使用结果 money_display 中的站内金额；写入 quota 参数仍为内部单位。"
		}
		if tool.MinRole == 0 {
			tool.MinRole = common.RoleAdminUser
		}
		if tool.Path != "" && tool.Method == "" {
			tool.Method = http.MethodGet
		}
		if tool.Permission.Resource == "" {
			switch tool.Category {
			case "channels":
				tool.Permission = authz.ChannelRead
			case "tickets":
				tool.Permission = authz.TicketAdminRead
			}
		}
		if tool.InputSchema == nil {
			tool.InputSchema = mcpToolInputSchema(*tool)
		}
		if tool.Annotations == nil {
			tool.Annotations = map[string]bool{
				"readOnlyHint":    !tool.Write,
				"destructiveHint": tool.Write,
				"idempotentHint":  true,
				"openWorldHint":   tool.Name == "newapi_channel_test" || tool.Name == "newapi_channel_balance_refresh",
			}
		}
	}
	return tools
}

// mcpToolInputSchema 将固定路径与业务 body 契约表达为可供客户端发现的 JSON Schema。
func mcpToolInputSchema(tool MCPTool) map[string]any {
	properties := map[string]any{}
	required := []string{}
	if tool.Path != "" {
		properties["query"] = map[string]any{
			"type": "object", "maxProperties": 30,
			"description":          "后台接口的查询参数，所有值使用字符串；分页每页最多 100 条，时间使用 Unix 秒。",
			"additionalProperties": map[string]any{"type": "string", "maxLength": 2048},
		}
	}
	pathProperties := map[string]any{}
	pathRequired := []string{}
	for _, segment := range strings.Split(tool.Path, "/") {
		if !strings.HasPrefix(segment, ":") {
			continue
		}
		name := strings.TrimPrefix(segment, ":")
		pathProperties[name] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "pattern": "^[A-Za-z0-9_-]+$"}
		pathRequired = append(pathRequired, name)
	}
	if len(pathRequired) > 0 {
		properties["path"] = map[string]any{"type": "object", "properties": pathProperties, "required": pathRequired, "additionalProperties": false}
		required = append(required, "path")
	}
	if tool.Write {
		properties["request_id"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "description": "本次操作唯一标识；重试保持标识与参数一致。未知状态先查询，不换标识重复执行。"}
		required = append(required, "request_id")
		if tool.Method == http.MethodPost || tool.Method == http.MethodPut || tool.Method == http.MethodPatch {
			properties["body"] = map[string]any{"type": "object", "description": tool.Description, "additionalProperties": true}
			required = append(required, "body")
		}
	}
	if tool.Name == "newapi_channel_status_set" {
		properties["body"] = map[string]any{"type": "object", "properties": map[string]any{"status": map[string]any{"type": "integer", "enum": []int{1, 2}}}, "required": []string{"status"}, "additionalProperties": false}
	}
	if tool.Name == "newapi_channel_update" {
		properties["body"] = map[string]any{
			"type": "object", "additionalProperties": false, "required": []string{"id"},
			"properties": map[string]any{
				"id":       map[string]any{"type": "integer", "minimum": 1},
				"name":     map[string]any{"type": "string", "maxLength": 255},
				"models":   map[string]any{"type": "string", "maxLength": 8192, "description": "完整模型 CSV，最多 8192 字节、512 项；先读取并合并原列表再保存。"},
				"group":    map[string]any{"type": "string", "maxLength": 4096},
				"priority": map[string]any{"type": "integer", "minimum": 0, "maximum": 2147483647},
				"weight":   map[string]any{"type": "integer", "minimum": 0, "maximum": 2147483647},
			},
		}
	}
	if tool.Name == "newapi_channel_create" {
		properties["body"] = map[string]any{
			"type": "object", "additionalProperties": false, "required": []string{"mode", "channel"},
			"properties": map[string]any{
				"mode":           map[string]any{"type": "string", "enum": []string{"single", "multi_to_single"}},
				"multi_key_mode": map[string]any{"type": "string", "enum": []string{"random", "polling"}},
				"channel": map[string]any{
					"type": "object", "additionalProperties": true,
					"required": []string{"name", "type", "key", "models", "group"},
					"properties": map[string]any{
						"name":   map[string]any{"type": "string", "minLength": 1, "maxLength": 255},
						"type":   map[string]any{"type": "integer", "minimum": 1, "maximum": 10000},
						"key":    map[string]any{"type": "string", "minLength": 1, "maxLength": 32768},
						"models": map[string]any{"type": "string", "minLength": 1, "maxLength": 8192},
						"group":  map[string]any{"type": "string", "minLength": 1, "maxLength": 4096},
					},
				},
			},
		}
	}
	if tool.Name == "newapi_ticket_status_set" {
		properties["body"] = map[string]any{"type": "object", "properties": map[string]any{"status": map[string]any{"type": "string", "enum": []string{"open", "processing", "resolved", "closed"}}}, "required": []string{"status"}, "additionalProperties": false}
	}
	switch tool.Name {
	case "newapi_model_create", "newapi_model_update":
		fields := map[string]any{
			"model_name":    map[string]any{"type": "string", "minLength": 1, "maxLength": 128},
			"description":   map[string]any{"type": "string", "maxLength": 8192},
			"icon":          map[string]any{"type": "string", "maxLength": 128},
			"tags":          map[string]any{"type": "string", "maxLength": 255},
			"vendor_id":     map[string]any{"type": "integer", "minimum": 0, "maximum": 2147483647, "description": "供应商 ID；0 表示不绑定供应商。"},
			"endpoints":     map[string]any{"type": "string", "maxLength": 8192, "description": "JSON 字符串，支持端点类型数组或端点名称到 path/method 对象的映射；无配置传空字符串。"},
			"status":        map[string]any{"type": "integer", "enum": []int{0, 1}, "description": "0 禁用，1 启用；创建时省略为 0。"},
			"sync_official": map[string]any{"type": "integer", "enum": []int{0, 1}, "description": "是否跟随官方元数据同步；创建时省略为 0。"},
			"name_rule":     map[string]any{"type": "integer", "enum": []int{0, 1, 2, 3}, "description": "0 精确、1 前缀、2 包含、3 后缀。"},
		}
		bodyRequired := []string{"model_name"}
		if tool.Name == "newapi_model_update" {
			fields["id"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483647}
			bodyRequired = []string{"id", "model_name", "description", "icon", "tags", "vendor_id", "endpoints", "status", "sync_official", "name_rule"}
		}
		properties["body"] = map[string]any{"type": "object", "properties": fields, "required": bodyRequired, "additionalProperties": false}
	case "newapi_vendor_create", "newapi_vendor_update":
		fields := map[string]any{
			"name":        map[string]any{"type": "string", "minLength": 1, "maxLength": 128},
			"description": map[string]any{"type": "string", "maxLength": 8192},
			"icon":        map[string]any{"type": "string", "maxLength": 128},
			"status":      map[string]any{"type": "integer", "enum": []int{0, 1}},
		}
		bodyRequired := []string{"name"}
		if tool.Name == "newapi_vendor_update" {
			fields["id"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483647}
			bodyRequired = []string{"id", "name", "description", "icon", "status"}
		}
		properties["body"] = map[string]any{"type": "object", "properties": fields, "required": bodyRequired, "additionalProperties": false}
	case "newapi_group_tag_create", "newapi_group_tag_update":
		fields := map[string]any{
			"name":           map[string]any{"type": "string", "minLength": 1, "maxLength": 64},
			"description":    map[string]any{"type": "string", "maxLength": 512},
			"icons":          map[string]any{"type": "array", "maxItems": 6, "items": map[string]any{"type": "string", "minLength": 1, "maxLength": 2048}},
			"sort_order":     map[string]any{"type": "integer", "minimum": -1000000, "maximum": 1000000},
			"group_ids":      map[string]any{"type": "array", "maxItems": 10000, "items": map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483647}},
			"match_keywords": map[string]any{"type": "array", "maxItems": 32, "items": map[string]any{"type": "string", "maxLength": 128}},
		}
		bodyRequired := []string{"name"}
		if tool.Name == "newapi_group_tag_update" {
			bodyRequired = []string{"name", "description", "icons", "sort_order", "group_ids", "match_keywords"}
		}
		properties["body"] = map[string]any{"type": "object", "properties": fields, "required": bodyRequired, "additionalProperties": false}
	case "newapi_channel_models":
		properties["query"] = map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"p":         map[string]any{"type": "string", "pattern": "^[0-9]+$", "maxLength": 5, "description": "页码，1–10000，默认 1。"},
				"page_size": map[string]any{"type": "string", "pattern": "^[0-9]+$", "maxLength": 3, "description": "每页条数，1–100，默认 20。"},
			},
		}
	case "newapi_channel_test":
		properties["query"] = map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"model":         map[string]any{"type": "string", "maxLength": 255, "description": "省略时使用渠道默认测试模型。"},
				"endpoint_type": map[string]any{"type": "string", "enum": []string{"", "openai", "openai-response", "openai-response-compact", "anthropic", "gemini", "jina-rerank", "image-generation", "embeddings"}, "description": "省略或空字符串由宿主选择协议；能力受渠道适配器限制。"},
				"stream":        map[string]any{"type": "string", "enum": []string{"true", "false"}, "description": "是否测试流式模式；只在宿主与上游支持的协议有效，返回最终测试摘要。"},
			},
		}
	}
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}
