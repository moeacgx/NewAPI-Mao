package service

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/ionet"
)

// 以下结构只描述可提交的管理合同，不从数据库模型推导写入字段。
type mcpNotificationTargetInput struct {
	ID            int    `json:"id,omitempty"`
	ChatID        string `json:"chat_id"`
	MentionUserID string `json:"mention_user_id,omitempty"`
	MentionName   string `json:"mention_name,omitempty"`
	Enabled       *bool  `json:"enabled,omitempty"`
}

type mcpNotificationFilterInput struct {
	StatusCodes        string   `json:"status_codes,omitempty"`
	ErrorKeywords      []string `json:"error_keywords,omitempty"`
	PrefixDedupSeconds int      `json:"prefix_dedup_seconds,omitempty"`
}

type mcpNotificationTaskInput struct {
	Name         string                       `json:"name"`
	EventType    string                       `json:"event_type"`
	BotID        int                          `json:"bot_id"`
	Template     string                       `json:"template"`
	Enabled      bool                         `json:"enabled"`
	FilterConfig *mcpNotificationFilterInput  `json:"filter_config,omitempty"`
	Targets      []mcpNotificationTargetInput `json:"targets"`
}

type mcpTaskPluginUploadInput struct {
	Source          string `json:"source"`
	SourceSHA256    string `json:"sourceSha256,omitempty"`
	ExpectedKey     string `json:"expectedKey,omitempty"`
	ExpectedVersion string `json:"expectedVersion,omitempty"`
	Remark          string `json:"remark,omitempty"`
	Marketplace     *struct {
		Name     string `json:"name"`
		IndexURL string `json:"index_url"`
		Path     string `json:"path"`
	} `json:"marketplace,omitempty"`
}

type mcpTaskPluginSourceInput struct {
	Name     string `json:"name"`
	IndexURL string `json:"index_url"`
}

type mcpRateConfigInput struct {
	RateAPIURL      string  `json:"rate_api_url"`
	Side            string  `json:"side"`
	Tier            int     `json:"tier"`
	AdjustmentType  string  `json:"adjustment_type"`
	AdjustmentValue float64 `json:"adjustment_value"`
}

// mcpFullOperationTools 只补充原目录未注册的名称；原查询的投影由独立 hook 补齐。
func mcpFullOperationTools() []MCPTool {
	tools := []MCPTool{
		{Name: "newapi_notification_tasks_edit_list", Category: "notifications", Path: "/api/notification/tasks", Description: "敏感配置读取：返回全部通知任务模板、接收目标及筛选，供完整编辑；与旧任务摘要工具分开授权。"},
		{Name: "newapi_notification_event_definitions", Category: "notifications", Path: "/api/notification/event-types", Description: "读取通知事件定义、默认模板和可用变量，供创建或编辑规则；不读取历史消息。"},
		{Name: "newapi_notification_bots", Category: "notifications", Path: "/api/notification/bots", Description: "查询 Bot 配置及 token_configured；不返回 Token。"},
		{Name: "newapi_notification_bot_create", Category: "notifications", Method: http.MethodPost, Path: "/api/notification/bots", Write: true, Description: "创建 Telegram Bot。token 仅作为输入；不自动发送测试消息。"},
		{Name: "newapi_notification_bot_update", Category: "notifications", Method: http.MethodPut, Path: "/api/notification/bots/:id", Write: true, Description: "更新 Bot；省略 token 保留现有凭据，enabled=false 会清理历史。"},
		{Name: "newapi_notification_bot_disable", Category: "notifications", Method: http.MethodDelete, Path: "/api/notification/bots/:id", Write: true, Description: "停用 Bot 并清理关联通知历史；宿主不物理删除 Bot。"},
		{Name: "newapi_notification_bot_test", Category: "notifications", Method: http.MethodPost, Path: "/api/notification/bots/:id/test", Write: true, Description: "向明确的 chat_id 发送真实 Telegram 测试消息，并更新检测状态。"},
		{Name: "newapi_notification_task_create", Category: "notifications", Method: http.MethodPost, Path: "/api/notification/tasks", Write: true, Description: "创建通知规则及接收目标。启用后会发送后续匹配事件；data 返回规则 ID。"},
		{Name: "newapi_notification_task_update", Category: "notifications", Method: http.MethodPut, Path: "/api/notification/tasks/:id", Write: true, Description: "完整替换通知规则，先读任务列表合并模板、筛选及 targets；省略 filter_config 清空筛选。"},
		{Name: "newapi_notification_task_disable", Category: "notifications", Method: http.MethodDelete, Path: "/api/notification/tasks/:id", Write: true, Description: "停用规则并取消待发送任务，清理关联跟踪；不物理删除规则。"},
		{Name: "newapi_notification_preview", Category: "notifications", Method: http.MethodPost, Path: "/api/notification/preview", Description: "敏感只读：仅使用宿主样例渲染通知 HTML，不发送消息，不保存正文回执。"},
		{Name: "newapi_extension_marketplace", Category: "extensions", Path: "/api/extension-admin/marketplace", Description: "查询固定 ZIP 市场来源、宿主版本与大小上限；安装可调用固定 Base64 ZIP 工具。"},
		{Name: "newapi_extension_notification_publish", Category: "extensions", Method: http.MethodPost, Path: "/api/extensions/:id/notification-events", Write: true, StringPathIDs: []string{"id"}, Description: "Root 独立授权发布已启用扩展的 manifest 通知事件，可能向配置目标发送消息。body 最多 16 KiB；payload 仅声明的标量变量，event_key 由宿主去重。返回队列状态及投递任务数量，queued 不代表已发送。"},
		{Name: "newapi_extensions_inventory", Category: "extensions", Path: "/api/extension-admin/", Description: "读取扩展清单、状态、权限声明和通知模板；query.all=true 包含停用模块，排除宿主路径与认证秘密。"},
		{Name: "newapi_extension_upload", Category: "extensions", Method: http.MethodPost, Path: "/api/extension-admin/upload", UploadField: "file", Write: true, Description: "安装或更新 ZIP 扩展：filename 必须为 .zip，content_base64 为标准 Base64，文件最多 6 MiB且外层 JSON 最多 8 MiB。固定映射宿主 file 文件字段；市场校验五字段须全部提供或全部省略。首次安装关闭，更新保留状态；更大 ZIP 使用后台上传。"},
		{Name: "newapi_extensions_refresh", Category: "extensions", Method: http.MethodPost, Path: "/api/extension-admin/refresh", Write: true, Description: "刷新当前宿主节点扩展注册表。多实例需分别刷新并核对，单次不代表集群完成。"},
		{Name: "newapi_extension_enabled_set", Category: "extensions", Method: http.MethodPut, Path: "/api/extension-admin/:id/enabled", Write: true, StringPathIDs: []string{"id"}, Description: "按模块字符串 ID 启停已安装扩展。"},
		{Name: "newapi_extension_uninstall", Category: "extensions", Method: http.MethodDelete, Path: "/api/extension-admin/:id", Write: true, StringPathIDs: []string{"id"}, Description: "按模块字符串 ID 卸载外部扩展；宿主限制仍生效。"},
		{Name: "newapi_extension_rate_config", Category: "extensions", Path: "/api/extension-admin/okx-alipay-rate/config", Description: "读取 OKX 支付宝汇率完整配置；带凭据的 URL 经脱敏，保存时需重新提供。"},
		{Name: "newapi_extension_rate_config_set", Category: "extensions", Method: http.MethodPut, Path: "/api/extension-admin/okx-alipay-rate/config", Write: true, Description: "完整保存汇率源、买卖方向、档位与调整值；URL 仍由宿主执行 SSRF 校验。"},
		{Name: "newapi_extension_rate_preview", Category: "extensions", Path: "/api/extension-admin/okx-alipay-rate/quote", Write: true, Description: "主动请求配置的汇率源，返回报价预览；不创建支付订单。"},
		{Name: "newapi_plugin_get", Category: "extensions", Path: "/api/plugin/task/:key", MinRole: common.RoleAdminUser, Description: "查询任务插件版本、能力声明和引用统计；query.version 可指定历史版本。不返回源码和认证配置。"},
		{Name: "newapi_plugin_source_get", Category: "extensions", Path: "/api/plugin/task/:key", Description: "Root 独立敏感授权：读取插件原始源码及版本/hash，query.version 可指定历史版本。源码可能含硬编码秘密；原样返回供编辑及校验，不保存写回执。普通插件详情授权不包含此能力。"},
		{Name: "newapi_plugins_inventory", Category: "extensions", Path: "/api/plugin/task", MinRole: common.RoleAdminUser, Description: "读取任务插件完整管理清单，包含业务 key、版本、声明、来源及引用统计；不含源码和认证配置。"},
		{Name: "newapi_plugin_versions", Category: "extensions", Path: "/api/plugin/task/:key/versions", MinRole: common.RoleAdminUser, Description: "查询归档版本、hash、活动状态与来源。"},
		{Name: "newapi_plugin_options", Category: "extensions", Path: "/api/task_plugin_options", MinRole: common.RoleAdminUser, Description: "查询渠道绑定所需插件 key、名称、版本、模型与渠道类型。"},
		{Name: "newapi_plugin_upload", Category: "extensions", Method: http.MethodPost, Path: "/api/plugin/task", Write: true, Description: "编译校验 JSON 源码并归档未激活、禁用版本；同 key/version 不得换源。MCP 总请求上限 1 MiB，大文件需后台上传。市场安装须完整提供 SHA256、预期 key/version 及 marketplace。"},
		{Name: "newapi_plugin_upload_put", Category: "extensions", Method: http.MethodPut, Path: "/api/plugin/task", Write: true, Description: "通过宿主 PUT 别名编译校验并归档插件源码；与 POST 上传合同一致，不允许覆盖同 key/version 的不同源码，不自动激活。"},
		{Name: "newapi_plugin_activate", Category: "extensions", Method: http.MethodPost, Path: "/api/plugin/task/:key/activate", Write: true, Description: "激活并启用指定插件版本；仍需总运行时开关开启及渠道绑定。"},
		{Name: "newapi_plugin_status_set", Category: "extensions", Method: http.MethodPost, Path: "/api/plugin/task/:key/status", Write: true, Description: "启停插件活动版本；历史固定版本任务按宿主生命周期处理。"},
		{Name: "newapi_plugin_version_delete", Category: "extensions", Method: http.MethodDelete, Path: "/api/plugin/task/:key/versions/:version", Write: true, Description: "仅删除非活动且没有历史任务引用的插件版本。"},
		{Name: "newapi_plugin_runtime_set", Category: "extensions", Method: http.MethodPut, Path: "/api/plugin/task/runtime/status", Write: true, Description: "设置任务插件总开关并刷新本节点路由；关闭不取消历史固定版本任务。"},
		{Name: "newapi_plugin_marketplace_sources", Category: "extensions", Path: "/api/plugin/task/marketplace/sources", MinRole: common.RoleAdminUser, Description: "读取任务插件 HTTPS 市场源；空数组表示显式清空。"},
		{Name: "newapi_plugin_marketplace_sources_set", Category: "extensions", Method: http.MethodPut, Path: "/api/plugin/task/marketplace/sources", Write: true, Description: "完整保存最多 16 个市场源，body 为裸数组；[] 清空配置，不移除已装插件。"},
		{Name: "newapi_deployments_list", Category: "deployments", Path: "/api/deployments/", Description: "分页查询 io.net 部署及状态计数；会访问供应商。"},
		{Name: "newapi_deployments_search", Category: "deployments", Path: "/api/deployments/search", Description: "按状态分页，再对当前页按 keyword 筛选部署名；total 遵循宿主当前页搜索口径。"},
		{Name: "newapi_deployment_get", Category: "deployments", Path: "/api/deployments/:id", StringPathIDs: []string{"id"}, Description: "查询供应商部署状态、计费时长和可编辑容器配置；环境变量仅返回配置标记，避免泄露隐含凭据。"},
		{Name: "newapi_deployment_create", Category: "deployments", Method: http.MethodPost, Path: "/api/deployments/", Write: true, Description: "在 io.net 创建付费部署；镜像认证和 secret_env_variables 仅输入。返回部署 ID 后继续查询状态，不把受理当完成。"},
		{Name: "newapi_deployment_update", Category: "deployments", Method: http.MethodPut, Path: "/api/deployments/:id", Write: true, StringPathIDs: []string{"id"}, Description: "更新容器配置，省略字段遵循供应商语义；环境变量被隐藏，不能用读取结果盲目覆盖。可能触发重新部署。"},
		{Name: "newapi_deployment_name_set", Category: "deployments", Method: http.MethodPut, Path: "/api/deployments/:id/name", Write: true, StringPathIDs: []string{"id"}, Description: "检查可用性并修改部署名称。"},
		{Name: "newapi_deployment_extend", Category: "deployments", Method: http.MethodPost, Path: "/api/deployments/:id/extend", Write: true, StringPathIDs: []string{"id"}, Description: "增加部署时长，会产生供应商费用。"},
		{Name: "newapi_deployment_delete", Category: "deployments", Method: http.MethodDelete, Path: "/api/deployments/:id", Write: true, StringPathIDs: []string{"id"}, Description: "请求终止部署；受理不等于容器已销毁，需继续查询状态。"},
		{Name: "newapi_deployment_test_connection", Category: "deployments", Method: http.MethodPost, Path: "/api/deployments/test-connection", Write: true, Description: "检测提供或已保存的 io.net API Key，返回硬件数量；不回传凭据。"},
		{Name: "newapi_deployment_settings_test_connection", Category: "deployments", Method: http.MethodPost, Path: "/api/deployments/settings/test-connection", Write: true, Description: "通过设置页检测接口验证提供或已保存的 io.net API Key，仅返回硬件和可用量计数。"},
		{Name: "newapi_deployment_logs", Category: "deployments", Path: "/api/deployments/:id/logs", MinRole: common.RoleRootUser, StringPathIDs: []string{"id"}, StringQueryIDs: []string{"container_id"}, Description: "敏感日志读取，需 Root 单独授权。必填 query.container_id；非流式最多 100 条、输出 16 KiB UTF-8 文本。含 credentials/headers/认证标记的段落整体隐藏至空行，返回 text/redacted/truncated；无标记秘密仍可能存在，不保存正文回执。"},
		{Name: "newapi_deployment_hardware_types", Category: "deployments", Path: "/api/deployments/hardware-types", Description: "查询可部署硬件及单价。"},
		{Name: "newapi_deployment_locations", Category: "deployments", Path: "/api/deployments/locations", Description: "查询部署位置及可用量。"},
		{Name: "newapi_deployment_replicas", Category: "deployments", Path: "/api/deployments/available-replicas", Description: "按 hardware_id 和可选 gpu_count 查询可用副本。"},
		{Name: "newapi_deployment_price_estimate", Category: "deployments", Method: http.MethodPost, Path: "/api/deployments/price-estimation", Write: true, Description: "请求供应商估价，不创建部署；使用 request_id 固定本次查询。"},
		{Name: "newapi_deployment_name_check", Category: "deployments", Path: "/api/deployments/check-name", Description: "检测 query.name 部署名称可用性。"},
		{Name: "newapi_deployment_containers", Category: "deployments", Path: "/api/deployments/:id/containers", StringPathIDs: []string{"id"}, Description: "查询部署容器状态；原始事件文本与 URL 凭据不返回。"},
		{Name: "newapi_deployment_container_get", Category: "deployments", Path: "/api/deployments/:id/containers/:container_id", StringPathIDs: []string{"id", "container_id"}, Description: "按供应商部署与容器字符串 ID 查询运行状态。"},
	}
	for i := range tools {
		t := &tools[i]
		t.TypedArguments = true
		if t.MinRole == 0 {
			t.MinRole = common.RoleRootUser
			if t.Category == "deployments" {
				t.MinRole = common.RoleAdminUser
			}
		}
		if t.Method == "" {
			t.Method = http.MethodGet
		}
		t.InputSchema = mcpToolInputSchema(*t)
		t.Annotations = map[string]bool{"readOnlyHint": !t.Write, "destructiveHint": t.Write, "idempotentHint": true, "openWorldHint": t.Category == "deployments" || t.Name == "newapi_notification_bot_test" || t.Name == "newapi_extension_rate_preview" || t.Name == "newapi_extension_notification_publish"}
		props := t.InputSchema["properties"].(map[string]any)
		props["query"] = mcpOperationQuerySchema("")
		if t.Method == http.MethodPost || t.Method == http.MethodPut {
			props["body"] = mcpFullOperationBodySchema(t.Name)
			if !t.Write {
				t.InputSchema["required"] = append(t.InputSchema["required"].([]string), "body")
			}
		}
		switch t.Name {
		case "newapi_extensions_inventory":
			props["query"] = mcpOperationQuerySchema("all")
			props["query"].(map[string]any)["properties"].(map[string]any)["all"].(map[string]any)["enum"] = []string{"true", "false"}
		case "newapi_plugin_get", "newapi_plugin_source_get":
			props["query"] = mcpOperationQuerySchema("version")
		case "newapi_deployments_list":
			props["query"] = mcpOperationQuerySchema("p page_size status")
		case "newapi_deployments_search":
			props["query"] = mcpOperationQuerySchema("p page_size status keyword")
		case "newapi_deployment_replicas":
			query := mcpOperationQuerySchema("hardware_id gpu_count")
			query["required"] = []string{"hardware_id"}
			props["query"] = query
			t.InputSchema["required"] = append(t.InputSchema["required"].([]string), "query")
		case "newapi_deployment_name_check":
			query := mcpOperationQuerySchema("name")
			query["required"] = []string{"name"}
			props["query"] = query
			t.InputSchema["required"] = append(t.InputSchema["required"].([]string), "query")
		case "newapi_deployment_logs":
			query := mcpOperationQuerySchema("container_id level stream limit cursor follow start_time end_time")
			fields := query["properties"].(map[string]any)
			fields["container_id"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "pattern": "^[A-Za-z0-9][A-Za-z0-9_-]*$"}
			fields["limit"] = map[string]any{"type": "string", "pattern": "^([1-9]|[1-9][0-9]|100)$"}
			fields["follow"] = map[string]any{"type": "string", "enum": []string{"false"}}
			// level/stream 是供应商过滤值；不假设宿主未声明的枚举，但拒绝控制字符。
			for _, field := range []string{"level", "stream"} {
				fields[field] = map[string]any{"type": "string", "minLength": 1, "maxLength": 32, "pattern": "^[A-Za-z0-9_-]+$"}
			}
			for _, field := range []string{"start_time", "end_time"} {
				fields[field] = map[string]any{"type": "string", "maxLength": 35, "pattern": `^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-][0-9]{2}:[0-9]{2})$`}
			}
			query["required"] = []string{"container_id"}
			props["query"] = query
			t.InputSchema["required"] = append(t.InputSchema["required"].([]string), "query")
		case "newapi_plugin_version_delete":
			path := props["path"].(map[string]any)["properties"].(map[string]any)
			path["version"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "pattern": "^[A-Za-z0-9][A-Za-z0-9._+-]*$"}
		}
	}
	return tools
}

func mcpOperationQuerySchema(fields string) map[string]any {
	props := map[string]any{}
	for _, field := range strings.Fields(fields) {
		props[field] = map[string]any{"type": "string", "minLength": 1, "maxLength": 2048}
		if field == "hardware_id" || field == "gpu_count" {
			props[field] = map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,8}$"}
		}
	}
	return map[string]any{"type": "object", "properties": props, "additionalProperties": false}
}

func mcpFullOperationBodySchema(name string) map[string]any {
	var input any = struct{}{}
	switch name {
	case "newapi_extension_notification_publish":
		schema := mcpTypedBodySchema(struct {
			EventType string         `json:"event_type"`
			EventKey  string         `json:"event_key"`
			Payload   map[string]any `json:"payload"`
		}{})
		fields := schema["properties"].(map[string]any)
		fields["event_type"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "pattern": `^[A-Za-z0-9][A-Za-z0-9._-]*$`}
		fields["event_key"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "pattern": `^[^\x00-\x1f\x7f]+$`}
		fields["payload"] = map[string]any{"type": "object", "maxProperties": 32, "additionalProperties": map[string]any{
			"type": []string{"string", "number", "boolean", "null"}, "maxLength": 1024, "minimum": -1e15, "maximum": 1e15,
		}}
		return schema
	case "newapi_notification_bot_create":
		input = struct {
			Name    string `json:"name"`
			Token   string `json:"token"`
			Enabled *bool  `json:"enabled,omitempty"`
		}{}
	case "newapi_notification_bot_update":
		input = struct {
			Name    string  `json:"name,omitempty"`
			Token   *string `json:"token,omitempty"`
			Enabled *bool   `json:"enabled,omitempty"`
		}{}
	case "newapi_notification_bot_test":
		input = struct {
			ChatID  string `json:"chat_id"`
			Message string `json:"message,omitempty"`
		}{}
	case "newapi_notification_task_create", "newapi_notification_task_update":
		input = mcpNotificationTaskInput{}
	case "newapi_notification_preview":
		input = struct {
			EventType string `json:"event_type"`
			Template  string `json:"template"`
			Target    *struct {
				MentionUserID string `json:"mention_user_id,omitempty"`
				MentionName   string `json:"mention_name,omitempty"`
			} `json:"target,omitempty"`
		}{}
	case "newapi_extension_enabled_set", "newapi_plugin_status_set", "newapi_plugin_runtime_set":
		input = struct {
			Enabled bool `json:"enabled"`
		}{}
	case "newapi_extension_rate_config_set":
		input = mcpRateConfigInput{}
	case "newapi_extension_upload":
		schema := mcpUploadSchema()
		fields := schema["properties"].(map[string]any)
		fields["filename"] = map[string]any{"type": "string", "minLength": 5, "maxLength": 128, "pattern": `^[A-Za-z0-9][A-Za-z0-9._-]{0,123}\.[zZ][iI][pP]$`}
		fields["archiveSha256"] = map[string]any{"type": "string", "pattern": "^[a-fA-F0-9]{64}$"}
		fields["expectedId"] = map[string]any{"type": "string", "pattern": "^[a-z0-9][a-z0-9_-]{0,63}$"}
		fields["expectedVersion"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 128, "pattern": "^[A-Za-z0-9][A-Za-z0-9._+-]*$"}
		fields["catalogUrl"] = map[string]any{"type": "string", "enum": []string{"https://raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json"}}
		fields["archivePath"] = map[string]any{"type": "string", "minLength": 1, "maxLength": 512}
		return schema
	case "newapi_plugin_upload", "newapi_plugin_upload_put":
		input = mcpTaskPluginUploadInput{}
	case "newapi_plugin_activate":
		input = struct {
			Version string `json:"version"`
		}{}
	case "newapi_plugin_marketplace_sources_set":
		input = []mcpTaskPluginSourceInput{}
	case "newapi_deployment_create":
		input = ionet.DeploymentRequest{}
	case "newapi_deployment_update":
		input = ionet.UpdateDeploymentRequest{}
	case "newapi_deployment_name_set":
		input = struct {
			Name string `json:"name"`
		}{}
	case "newapi_deployment_extend":
		input = ionet.ExtendDurationRequest{}
	case "newapi_deployment_test_connection", "newapi_deployment_settings_test_connection":
		input = struct {
			APIKey string `json:"api_key,omitempty"`
		}{}
	case "newapi_deployment_price_estimate":
		input = ionet.PriceEstimationRequest{}
	}
	schema := mcpTypedBodySchema(input)
	mcpBoundOperationSchema(schema, "")
	if name == "newapi_plugin_marketplace_sources_set" {
		schema["maxItems"] = 16
	}
	if name == "newapi_extension_rate_config_set" {
		fields := schema["properties"].(map[string]any)
		fields["side"].(map[string]any)["enum"] = []string{"buy", "sell"}
		fields["adjustment_type"].(map[string]any)["enum"] = []string{"absolute", "percent"}
	}
	if name == "newapi_plugin_upload" || name == "newapi_plugin_upload_put" {
		schema["properties"].(map[string]any)["source"].(map[string]any)["maxLength"] = 1024 * 1024
	}
	return schema
}

// 每个计数和身份字段在请求 schema 内约束，避免全局 ID 规则误伤外部字符串 ID。
func mcpBoundOperationSchema(schema map[string]any, field string) {
	if schema["type"] == "integer" {
		minimum := 0
		if field == "bot_id" || field == "hardware_id" || field == "location_ids" || field == "duration_hours" || field == "gpus_per_container" || field == "replica_count" || field == "tier" {
			minimum = 1
		}
		schema["minimum"] = minimum
		schema["maximum"] = 2147483647
		switch field {
		case "duration_hours":
			schema["maximum"] = 87600
		case "gpus_per_container", "replica_count", "hardware_qty", "duration_qty":
			schema["maximum"] = 10000
		case "traffic_port":
			schema["maximum"] = 65535
		case "prefix_dedup_seconds":
			schema["maximum"] = 86400
		}
	}
	if schema["type"] == "string" {
		switch field {
		case "name", "chat_id", "mention_user_id", "mention_name", "expectedKey", "expectedVersion", "version":
			schema["maxLength"] = 128
		case "event_type":
			schema["maxLength"] = 64
		case "remark":
			schema["maxLength"] = 1024
		case "source":
			schema["minLength"] = 1
			schema["maxLength"] = 60000
		}
	}
	if schema["type"] == "array" {
		schema["maxItems"] = 100
		if field == "targets" {
			schema["minItems"] = 1
		}
		if field == "error_keywords" {
			schema["maxItems"] = 64
		}
		mcpBoundOperationSchema(schema["items"].(map[string]any), field)
	}
	if properties, ok := schema["properties"].(map[string]any); ok {
		for key, child := range properties {
			mcpBoundOperationSchema(child.(map[string]any), key)
		}
	}
}
