package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
)

// mcpFullChannelTools 补齐运维能力；主目录追加后须保留这里的精确 schema。
func mcpFullChannelTools() []MCPTool {
	tools := []MCPTool{
		{Name: "newapi_channels_create_batch", Category: "channels", Method: http.MethodPost, Path: "/api/channel/", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "按共同配置批量创建独立渠道，mode 固定 batch。channel.key 为换行分隔密钥；Vertex 非 API key 模式为 JSON 数组。最多 1000 个密钥，group_ids 使用稳定 ID。禁止将密钥前缀加入名称；响应只返回成功状态，不返回新 ID，随后按名称查询。重试使用同一 request_id。"},
		{Name: "newapi_channel_config_get", Category: "channels", Path: "/api/channel/:id", Permission: authz.ChannelSensitiveWrite, Description: "读取可编辑渠道配置，需独立敏感权限。key 永不返回；redacted_fields 标识含凭据而整体隐藏的字段，隐藏值不可回写。修改安全字段时保留其他配置，隐藏字段省略或提供完整新值。"},
		{Name: "newapi_channel_config_update", Category: "channels", Method: http.MethodPut, Path: "/api/channel/", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "更新渠道完整可编辑配置；先用 config_get 读取，按需提交。提交 setting 时带原 type 供宿主校验；高级自定义、Vertex、插件渠道还需保留相应 settings/other/base_url。省略字段沿用宿主部分更新语义，隐藏字段必须省略或提供完整新值。状态使用独立工具。models/group 是完整替换列表。"},
		{Name: "newapi_channels_test_all", Category: "channels", Path: "/api/channel/test", Write: true, Permission: authz.ChannelOperate, Description: "提交全渠道测试后台任务，可能产生上游费用并发送宿主已配置通知。返回 task_id；不代表测试已完成。"},
		{Name: "newapi_channels_balance_refresh", Category: "channels", Path: "/api/channel/update_balance", Write: true, Permission: authz.ChannelOperate, Description: "同步刷新宿主支持的启用渠道余额，跳过多密钥渠道；余额耗尽时可能自动停用渠道。逐个访问上游，可能耗时。"},
		{Name: "newapi_channels_status_set", Category: "channels", Method: http.MethodPost, Path: "/api/channel/status/batch", Write: true, Permission: authz.ChannelOperate, Description: "批量启用或手动停用指定渠道。"},
		{Name: "newapi_channels_delete", Category: "channels", Method: http.MethodPost, Path: "/api/channel/batch", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "批量删除指定渠道，不可撤销。"},
		{Name: "newapi_channels_disabled_delete", Category: "channels", Method: http.MethodDelete, Path: "/api/channel/disabled", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "删除所有已禁用渠道，不可撤销。"},
		{Name: "newapi_channels_abilities_repair", Category: "channels", Method: http.MethodPost, Path: "/api/channel/fix", Write: true, Permission: authz.ChannelOperate, Description: "修复渠道能力表并刷新缓存。"},
		{Name: "newapi_channel_models_fetch", Category: "channels", Path: "/api/channel/fetch_models/:id", Write: true, Permission: authz.ChannelOperate, Description: "使用已保存凭据请求渠道实时上游模型列表，不自动修改渠道 models。"},
		{Name: "newapi_channel_models_fetch_preview", Category: "channels", Method: http.MethodPost, Path: "/api/channel/fetch_models", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "按待保存配置拉取模型；输入凭据不回显。channel_id 仅适用于高级自定义渠道复用已保存配置。"},
		{Name: "newapi_channel_models_detect", Category: "channels", Method: http.MethodPost, Path: "/api/channel/upstream_updates/detect", Write: true, Permission: authz.ChannelOperate, Description: "检测一个渠道的上游模型变化并保存检测状态；返回待添加与待删除列表。"},
		{Name: "newapi_channels_models_detect", Category: "channels", Method: http.MethodPost, Path: "/api/channel/upstream_updates/detect_all", Write: true, Permission: authz.ChannelOperate, Description: "提交全部渠道模型检测后台任务，返回任务 ID；可能发送宿主已配置通知。"},
		{Name: "newapi_channel_models_apply", Category: "channels", Method: http.MethodPost, Path: "/api/channel/upstream_updates/apply", Write: true, Permission: authz.ChannelWrite, Description: "应用已检测的模型变化：add_models/remove_models/ignore_models 仅作用于当前待处理列表。"},
		{Name: "newapi_channels_models_apply", Category: "channels", Method: http.MethodPost, Path: "/api/channel/upstream_updates/apply_all", Write: true, Permission: authz.ChannelWrite, Description: "将所有启用且开启模型检测的渠道待添加、待删除变化全部应用。先检测并审阅变化。"},
		{Name: "newapi_channel_copy", Category: "channels", Method: http.MethodPost, Path: "/api/channel/copy/:id", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "复制渠道及凭据；query.suffix 默认 _复制，reset_balance 默认 true。只返回新渠道 ID。"},
		{Name: "newapi_channels_tag_set", Category: "channels", Method: http.MethodPost, Path: "/api/channel/batch/tag", Write: true, Permission: authz.ChannelWrite, Description: "批量设置渠道标签，tag 为空字符串时清空标签。"},
		{Name: "newapi_channel_tag_enable", Category: "channels", Method: http.MethodPost, Path: "/api/channel/tag/enabled", Write: true, Permission: authz.ChannelOperate, Description: "启用指定标签下的全部渠道。"},
		{Name: "newapi_channel_tag_disable", Category: "channels", Method: http.MethodPost, Path: "/api/channel/tag/disabled", Write: true, Permission: authz.ChannelOperate, Description: "停用指定标签下的全部渠道。"},
		{Name: "newapi_channel_tag_update", Category: "channels", Method: http.MethodPut, Path: "/api/channel/tag", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "批量更新指定标签渠道的路由和覆盖配置；models/groups 是完整覆盖，参数和请求头覆盖须敏感写权限。"},
		{Name: "newapi_channel_tag_models", Category: "channels", Path: "/api/channel/tag/models", Permission: authz.ChannelRead, Description: "读取标签下模型数量最多的渠道模型 CSV；不是全部渠道的模型并集。"},
		{Name: "newapi_channel_priority_reorder", Category: "channels", Method: http.MethodPost, Path: "/api/channel/priority/reorder", Write: true, Permission: authz.ChannelWrite, Description: "按 revision 快照调整优先级；position=before/after/same，preview=true 只预览。先查询优先级取得 revision。"},
		{Name: "newapi_channel_priority_restore", Category: "channels", Method: http.MethodPost, Path: "/api/channel/priority/reorder", Write: true, Permission: authz.ChannelWrite, Description: "按当前 revision 恢复优先级；restore 项的 previous 填当前优先级，priority 填期望恢复值，空值省略。撤销上次 changes 时交换 previous 与 priority。"},
		{Name: "newapi_channel_keys_status", Category: "channels", Method: http.MethodPost, Path: "/api/channel/multi_key/manage", Permission: authz.ChannelOperate, Description: "分页读取多密钥索引与状态；body.action 固定 get_key_status，不返回密钥预览或禁用原因原文。"},
		{Name: "newapi_channel_key_enable", Category: "channels", Method: http.MethodPost, Path: "/api/channel/multi_key/manage", Write: true, Permission: authz.ChannelOperate, Description: "启用指定从 0 开始的密钥索引，action 固定 enable_key。"},
		{Name: "newapi_channel_key_disable", Category: "channels", Method: http.MethodPost, Path: "/api/channel/multi_key/manage", Write: true, Permission: authz.ChannelOperate, Description: "停用指定从 0 开始的密钥索引，action 固定 disable_key。"},
		{Name: "newapi_channel_keys_enable", Category: "channels", Method: http.MethodPost, Path: "/api/channel/multi_key/manage", Write: true, Permission: authz.ChannelOperate, Description: "启用渠道全部密钥，action 固定 enable_all_keys。"},
		{Name: "newapi_channel_keys_disable", Category: "channels", Method: http.MethodPost, Path: "/api/channel/multi_key/manage", Write: true, Permission: authz.ChannelOperate, Description: "停用渠道全部密钥，action 固定 disable_all_keys。"},
		{Name: "newapi_channel_key_delete", Category: "channels", Method: http.MethodPost, Path: "/api/channel/multi_key/manage", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "删除指定密钥，后续索引会重排；必须重新读取状态。action 固定 delete_key。"},
		{Name: "newapi_channel_keys_disabled_delete", Category: "channels", Method: http.MethodPost, Path: "/api/channel/multi_key/manage", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "删除全部已禁用密钥，action 固定 delete_disabled_keys；宿主保留至少一个密钥。"},
		{Name: "newapi_channel_ollama_pull", Category: "channels", Method: http.MethodPost, Path: "/api/channel/ollama/pull", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "非流式拉取 Ollama 模型，可能耗时并占用上游磁盘；超时后先核对上游状态，不换 request_id 重复拉取。"},
		{Name: "newapi_channel_ollama_delete", Category: "channels", Method: http.MethodDelete, Path: "/api/channel/ollama/delete", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "删除上游 Ollama 模型；DELETE 请求携带 channel_id/model_name JSON body。"},
		{Name: "newapi_channel_ollama_version", Category: "channels", Path: "/api/channel/ollama/version/:id", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "主动请求 Ollama 服务版本，使用已保存凭据。"},
		{Name: "newapi_channel_codex_refresh", Category: "channels", Method: http.MethodPost, Path: "/api/channel/:id/codex/refresh", Write: true, Permission: authz.ChannelSensitiveWrite, Description: "刷新并保存 Codex OAuth 凭据，只返回刷新时间、有效期与渠道标识，不回显账户身份或令牌。"},
		{Name: "newapi_channel_codex_usage", Category: "channels", Path: "/api/channel/:id/codex/usage", Write: true, Permission: authz.ChannelRead, Description: "请求 Codex 上游额度窗口；上游鉴权失效时宿主可能自动刷新并保存凭据，因此要求 request_id。"},
		{Name: "newapi_channel_codex_reset_credits", Category: "channels", Path: "/api/channel/:id/codex/usage/reset-credits", Write: true, Permission: authz.ChannelRead, Description: "请求 Codex 可用重置次数与状态；可能自动刷新凭据，要求 request_id，不返回账户身份。"},
		{Name: "newapi_channel_codex_usage_reset", Category: "channels", Method: http.MethodPost, Path: "/api/channel/:id/codex/usage/reset", Write: true, Permission: authz.ChannelOperate, Description: "消耗 Codex 上游重置次数以重置额度窗口，重试必须使用同一个 request_id。"},
		{Name: "newapi_models_sync_preview", Category: "models", Path: "/api/models/sync_upstream/preview", Description: "从上游元数据源预览缺失模型及字段冲突；不更改本地模型，不同步价格。"},
		{Name: "newapi_models_sync", Category: "models", Method: http.MethodPost, Path: "/api/models/sync_upstream", Write: true, Description: "同步缺失模型与供应商元数据；overwrite 仅覆盖 sync_official 已启用模型的指定字段，不配置渠道或价格。"},
		{Name: "newapi_vendors_search", Category: "models", Path: "/api/vendors/search", Description: "搜索供应商元数据；支持 keyword 和 p/page_size 分页。"},
		{Name: "newapi_groups_save", Category: "groups", Method: http.MethodPut, Path: "/api/group/details", Write: true, Description: "保存分组完整编辑属性；每项必须提供非空 code，禁止 auto/all/null、空白或逗号。已有项携带稳定 id 并保留读取的 code；新建项省略 id，提供不与现有标识重复的临时 code，宿主保存后生成正式数字 code。deleted_ids 明确删除并迁移关联令牌；name 是显示名称。"},
		{Name: "newapi_groups_code_migration_preview", Category: "groups", Method: http.MethodPost, Path: "/api/group/code-migration/preview", Description: "预览历史分组 code 迁移为稳定数字 ID 的影响和阻塞项，不执行迁移。"},
		{Name: "newapi_groups_code_migrate", Category: "groups", Method: http.MethodPost, Path: "/api/group/code-migration", Write: true, Description: "执行历史分组标识迁移；先预览，再提交 confirm=true。"},
		{Name: "newapi_groups_token_migration_preview", Category: "groups", Method: http.MethodPost, Path: "/api/group/token-migration/preview", Description: "预览令牌从源分组迁往目标分组或 auto；auto 模式省略 target_group_id。"},
		{Name: "newapi_groups_tokens_migrate", Category: "groups", Method: http.MethodPost, Path: "/api/group/token-migration", Write: true, Description: "迁移源分组绑定令牌；target_group_mode=auto 会清除其他分组与倍率保护，先预览影响。"},
		{Name: "newapi_prefill_group_create", Category: "groups", Method: http.MethodPost, Path: "/api/prefill_group/", Write: true, Description: "创建模型、标签或端点预填集合；items 为字符串数组。"},
		{Name: "newapi_prefill_group_update", Category: "groups", Method: http.MethodPut, Path: "/api/prefill_group/", Write: true, Description: "完整保存预填集合；提供 id/name/type/items/description。"},
		{Name: "newapi_prefill_group_delete", Category: "groups", Method: http.MethodDelete, Path: "/api/prefill_group/:id", Write: true, Description: "删除预填集合，不删除实际渠道、模型或分组。"},
	}
	for i := range tools {
		t := &tools[i]
		t.MinRole = common.RoleAdminUser
		t.TypedArguments = true
		if t.Method == "" {
			t.Method = http.MethodGet
		}
		t.InputSchema = mcpToolInputSchema(*t)
		props := t.InputSchema["properties"].(map[string]any)
		props["query"] = mcpChannelObject(nil)
		body := mcpChannelObject(nil)
		switch t.Name {
		case "newapi_channel_config_update", "newapi_channels_create_batch":
			body = mcpChannelObject(map[string]any{
				"id": mcpChannelInt(1, 2147483647), "type": mcpChannelInt(1, constant.ChannelTypeDummy-1), "vendor_id": map[string]any{"type": []string{"integer", "null"}, "minimum": 0, "maximum": int64(2147483647)},
				"name": mcpChannelText(1, 255), "key": mcpChannelText(0, 32768), "base_url": mcpChannelText(0, 2048),
				"openai_organization": mcpChannelText(0, 255), "test_model": mcpChannelText(0, 255),
				"models": mcpChannelText(1, 8192), "group": mcpChannelText(1, 4096), "group_ids": mcpChannelArray(mcpChannelInt(1, 2147483647), 1, 1000),
				"priority": mcpChannelInt(0, 2147483647), "weight": mcpChannelInt(0, 2147483647), "concurrency_limit": mcpChannelInt(0, 2147483647),
				"model_mapping": mcpChannelText(0, 16384), "status_code_mapping": mcpChannelText(0, 1024), "auto_ban": mcpChannelEnumInt(0, 1),
				"other": mcpChannelText(0, 8192), "tag": mcpChannelText(0, 255), "remark": mcpChannelText(0, 255),
				"setting": mcpChannelText(0, 16384), "settings": mcpChannelText(0, 32768), "param_override": mcpChannelText(0, 16384), "header_override": mcpChannelText(0, 16384),
				"multi_key_mode": mcpChannelEnum("random", "polling"), "key_mode": mcpChannelEnum("append", "replace"), "convert_to_multi_key": map[string]any{"type": "boolean"},
			}, "id")
			if t.Name == "newapi_channels_create_batch" {
				fields := body["properties"].(map[string]any)
				for _, field := range []string{"id", "multi_key_mode", "key_mode", "convert_to_multi_key"} {
					delete(fields, field)
				}
				fields["key"] = mcpChannelText(1, 32768)
				fields["status"] = mcpChannelEnumInt(1, 2)
				body["required"] = []string{"type", "name", "key", "models", "group_ids"}
				body = mcpChannelObject(map[string]any{
					"mode": mcpChannelEnum("batch"), "channel": body,
					"batch_add_set_key_prefix_2_name": map[string]any{"type": "boolean", "enum": []bool{false}},
				}, "mode", "channel")
			}
		case "newapi_channels_status_set":
			body = mcpChannelObject(map[string]any{"ids": mcpChannelArray(mcpChannelInt(1, 2147483647), 1, 1000), "status": mcpChannelEnumInt(1, 2)}, "ids", "status")
		case "newapi_channels_delete", "newapi_channels_tag_set":
			fields := map[string]any{"ids": mcpChannelArray(mcpChannelInt(1, 2147483647), 1, 1000)}
			body = mcpChannelObject(fields, "ids")
			if t.Name == "newapi_channels_tag_set" {
				fields["tag"] = mcpChannelText(0, 255)
				body["required"] = []string{"ids", "tag"}
			}
		case "newapi_channel_models_fetch_preview":
			body = mcpChannelObject(map[string]any{"channel_id": mcpChannelInt(1, 2147483647), "type": mcpChannelInt(1, constant.ChannelTypeDummy-1), "base_url": mcpChannelText(0, 2048), "key": mcpChannelText(0, 32768), "advanced_custom": mcpChannelText(0, 32768), "header_override": mcpChannelText(0, 16384), "proxy": mcpChannelText(0, 2048)}, "type")
		case "newapi_channel_models_detect":
			body = mcpChannelObject(map[string]any{"id": mcpChannelInt(1, 2147483647)}, "id")
		case "newapi_channel_models_apply":
			body = mcpChannelObject(map[string]any{"id": mcpChannelInt(1, 2147483647), "add_models": mcpChannelArray(mcpChannelText(1, 255), 0, 512), "remove_models": mcpChannelArray(mcpChannelText(1, 255), 0, 512), "ignore_models": mcpChannelArray(mcpChannelText(1, 255), 0, 512)}, "id", "add_models", "remove_models", "ignore_models")
		case "newapi_channel_copy":
			props["query"] = mcpChannelObject(map[string]any{"suffix": mcpChannelText(0, 128), "reset_balance": mcpChannelEnum("true", "false")})
		case "newapi_channel_tag_enable", "newapi_channel_tag_disable", "newapi_channel_tag_update":
			fields := map[string]any{"tag": mcpChannelText(1, 255)}
			if t.Name == "newapi_channel_tag_update" {
				fields["new_tag"] = mcpChannelText(0, 255)
				for _, field := range []string{"priority", "weight", "concurrency_limit"} {
					fields[field] = mcpChannelInt(0, 2147483647)
				}
				for _, field := range []string{"model_mapping", "param_override", "header_override"} {
					fields[field] = mcpChannelText(0, 16384)
				}
				fields["models"] = mcpChannelText(1, 8192)
				fields["groups"] = mcpChannelText(1, 4096)
			}
			body = mcpChannelObject(fields, "tag")
		case "newapi_channel_tag_models":
			props["query"] = mcpChannelObject(map[string]any{"tag": mcpChannelText(1, 255)}, "tag")
			t.InputSchema["required"] = []string{"query"}
		case "newapi_channel_priority_reorder", "newapi_channel_priority_restore":
			fields := map[string]any{"revision": map[string]any{"type": "string", "pattern": "^[0-9a-f]{64}$"}, "preview": map[string]any{"type": "boolean"}}
			body = mcpChannelObject(fields, "revision")
			if t.Name == "newapi_channel_priority_reorder" {
				fields["id"], fields["target_id"] = mcpChannelInt(1, 2147483647), mcpChannelInt(1, 2147483647)
				fields["position"] = mcpChannelEnum("before", "after", "same")
				body["required"] = []string{"revision", "id", "target_id", "position"}
			} else {
				fields["restore"] = mcpChannelArray(mcpChannelObject(map[string]any{"id": mcpChannelInt(1, 2147483647), "previous": mcpChannelInt(-9007199254740991, 9007199254740991), "priority": mcpChannelInt(-9007199254740991, 9007199254740991)}, "id"), 1, 1000)
				body["required"] = []string{"revision", "restore"}
			}
		case "newapi_channel_keys_status", "newapi_channel_key_enable", "newapi_channel_key_disable", "newapi_channel_keys_enable", "newapi_channel_keys_disable", "newapi_channel_key_delete", "newapi_channel_keys_disabled_delete":
			actions := map[string]string{"newapi_channel_keys_status": "get_key_status", "newapi_channel_key_enable": "enable_key", "newapi_channel_key_disable": "disable_key", "newapi_channel_keys_enable": "enable_all_keys", "newapi_channel_keys_disable": "disable_all_keys", "newapi_channel_key_delete": "delete_key", "newapi_channel_keys_disabled_delete": "delete_disabled_keys"}
			fields := map[string]any{"channel_id": mcpChannelInt(1, 2147483647), "action": mcpChannelEnum(actions[t.Name])}
			body = mcpChannelObject(fields, "channel_id", "action")
			if strings.Contains(t.Name, "_key_") {
				fields["key_index"] = mcpChannelInt(0, 2147483647)
				body["required"] = []string{"channel_id", "action", "key_index"}
			}
			if t.Name == "newapi_channel_keys_status" {
				fields["page"], fields["page_size"], fields["status"] = mcpChannelInt(1, 10000), mcpChannelInt(1, 100), mcpChannelEnumInt(1, 2, 3)
			}
		case "newapi_channel_ollama_pull", "newapi_channel_ollama_delete":
			body = mcpChannelObject(map[string]any{"channel_id": mcpChannelInt(1, 2147483647), "model_name": mcpChannelText(1, 255)}, "channel_id", "model_name")
		case "newapi_models_sync_preview":
			props["query"] = mcpChannelObject(map[string]any{"locale": mcpChannelEnum("", "en", "zh-CN", "zh-TW", "ja")})
		case "newapi_vendors_search":
			props["query"] = mcpChannelObject(map[string]any{"keyword": mcpChannelText(0, 128), "p": map[string]any{"type": "string", "pattern": "^[0-9]+$", "maxLength": 5}, "page_size": map[string]any{"type": "string", "pattern": "^[0-9]+$", "maxLength": 3}})
		case "newapi_models_sync":
			body = mcpChannelObject(map[string]any{"locale": mcpChannelEnum("", "en", "zh-CN", "zh-TW", "ja"), "overwrite": mcpChannelArray(mcpChannelObject(map[string]any{"model_name": mcpChannelText(1, 255), "fields": mcpChannelArray(mcpChannelEnum("description", "icon", "tags", "vendor", "name_rule", "status"), 1, 6)}, "model_name", "fields"), 0, 512)})
		case "newapi_groups_save":
			group := mcpChannelObject(map[string]any{"id": mcpChannelInt(1, 2147483647), "code": mcpChannelText(1, 64), "name": mcpChannelText(1, 128), "description": mcpChannelText(0, 8192), "ratio": map[string]any{"type": "number", "minimum": 0, "maximum": 1000000}, "status": mcpChannelEnumInt(0, 1), "user_selectable": map[string]any{"type": "boolean"}, "exclusive": map[string]any{"type": "boolean"}, "single_user_concurrency_limit": mcpChannelInt(0, 2147483647), "auto_enabled": map[string]any{"type": "boolean"}, "auto_order": mcpChannelInt(0, 2147483647)}, "code", "name", "description", "ratio", "status", "user_selectable", "exclusive", "single_user_concurrency_limit", "auto_enabled", "auto_order")
			options := map[string]any{}
			for _, key := range []string{"AutoGroupConfig", "DefaultUseAutoGroup", "MaxTokenAutoGroups", "GroupGroupRatio", "group_ratio_setting.group_group_ratio", "TopupGroupRatio"} {
				options[key] = mcpChannelText(0, 16384)
			}
			body = mcpChannelObject(map[string]any{"groups": mcpChannelArray(group, 0, 1000), "deleted_ids": mcpChannelArray(mcpChannelInt(1, 2147483647), 0, 1000), "option_updates": mcpChannelObject(options), "auto_group": mcpChannelObject(map[string]any{"user_selectable": map[string]any{"type": "boolean"}, "description": mcpChannelText(0, 8192)}, "user_selectable", "description")}, "groups", "deleted_ids")
		case "newapi_groups_code_migrate":
			body = mcpChannelObject(map[string]any{"confirm": map[string]any{"type": "boolean", "enum": []bool{true}}}, "confirm")
		case "newapi_groups_token_migration_preview", "newapi_groups_tokens_migrate":
			body = mcpChannelObject(map[string]any{"source_group_id": mcpChannelInt(1, 2147483647), "target_group_id": mcpChannelInt(1, 2147483647), "target_group_mode": mcpChannelEnum("explicit", "auto")}, "source_group_id", "target_group_mode")
		case "newapi_prefill_group_create", "newapi_prefill_group_update":
			fields := map[string]any{"name": mcpChannelText(1, 64), "type": mcpChannelEnum("model", "tag", "endpoint"), "items": mcpChannelArray(mcpChannelText(1, 255), 0, 512), "description": mcpChannelText(0, 255)}
			body = mcpChannelObject(fields, "name", "type", "items", "description")
			if t.Name == "newapi_prefill_group_update" {
				fields["id"] = mcpChannelInt(1, 2147483647)
				body["required"] = []string{"id", "name", "type", "items", "description"}
			}
		}
		// POST 预览与查询同样有 body，但不因此要求写入 request_id。
		if t.Method == http.MethodPost || t.Method == http.MethodPut || t.Name == "newapi_channel_ollama_delete" {
			props["body"] = body
			required := []string{"body"}
			if _, ok := props["path"]; ok {
				required = append(required, "path")
			}
			if t.Write {
				required = append(required, "request_id")
			}
			t.InputSchema["required"] = required
		}
		openWorld := strings.Contains(t.Name, "fetch") || strings.Contains(t.Name, "detect") || strings.Contains(t.Name, "ollama") || strings.Contains(t.Name, "codex") || strings.Contains(t.Name, "sync") || t.Name == "newapi_channels_test_all" || t.Name == "newapi_channels_balance_refresh"
		t.Annotations = map[string]bool{"readOnlyHint": !t.Write, "destructiveHint": t.Write, "idempotentHint": true, "openWorldHint": openWorld}
	}
	return tools
}

func mcpChannelObject(fields map[string]any, required ...string) map[string]any {
	if fields == nil {
		fields = map[string]any{}
	}
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": fields, "required": required, "additionalProperties": false}
}

func mcpChannelText(minimum, maximum int) map[string]any {
	return map[string]any{"type": "string", "minLength": minimum, "maxLength": maximum}
}

func mcpChannelInt(minimum, maximum int64) map[string]any {
	return map[string]any{"type": "integer", "minimum": minimum, "maximum": maximum}
}

func mcpChannelEnum(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}

func mcpChannelEnumInt(values ...int) map[string]any {
	return map[string]any{"type": "integer", "enum": values}
}

func mcpChannelArray(items map[string]any, minimum, maximum int) map[string]any {
	return map[string]any{"type": "array", "items": items, "minItems": minimum, "maxItems": maximum}
}

// validateMCPFullChannelArguments 在 schema 校验后、获取幂等执行权前调用。
func validateMCPFullChannelArguments(tool MCPTool, body []byte) error {
	var fields map[string]json.RawMessage
	if len(body) == 0 {
		return nil
	}
	if err := common.Unmarshal(body, &fields); err != nil {
		return err
	}
	switch tool.Name {
	case "newapi_channels_create_batch":
		var channel map[string]json.RawMessage
		if err := common.Unmarshal(fields["channel"], &channel); err != nil {
			return err
		}
		// 与配置更新共享敏感标记和可编辑元数据检查，不改变实际发往宿主的 JSON。
		if err := validateMCPFullChannelArguments(MCPTool{Name: "newapi_channel_config_update"}, fields["channel"]); err != nil {
			return err
		}
		var key string
		if common.Unmarshal(channel["key"], &key) != nil || strings.TrimSpace(key) == "" {
			return errors.New("batch channel keys cannot be empty")
		}
		var channelType int
		_ = common.Unmarshal(channel["type"], &channelType)
		count := 0
		if channelType == constant.ChannelTypeVertexAi && strings.HasPrefix(strings.TrimSpace(key), "[") {
			var keys []json.RawMessage
			if common.UnmarshalJsonStr(key, &keys) != nil {
				return errors.New("invalid Vertex batch keys")
			}
			count = len(keys)
			for _, raw := range keys {
				switch common.GetJsonType(raw) {
				case "object":
				case "string":
					var text string
					if common.Unmarshal(raw, &text) != nil || strings.TrimSpace(text) == "" {
						return errors.New("invalid Vertex batch key")
					}
				default:
					return errors.New("invalid Vertex batch key")
				}
			}
		} else {
			for _, line := range strings.Split(key, "\n") {
				if line == "" {
					continue
				}
				if strings.TrimSpace(line) == "" {
					return errors.New("batch channel key cannot contain a whitespace-only line")
				}
				count++
			}
		}
		if count < 1 || count > 1000 {
			return errors.New("batch channel key count must be between 1 and 1000")
		}
	case "newapi_channel_config_update", "newapi_channel_tag_update":
		for _, field := range []string{"key", "base_url", "openai_organization", "other", "setting", "settings", "model_mapping", "status_code_mapping", "param_override", "header_override"} {
			var value string
			if raw, ok := fields[field]; ok && common.Unmarshal(raw, &value) == nil && strings.Contains(value, "[redacted]") {
				return errors.New("redacted channel configuration must be omitted or replaced with a complete value")
			}
		}
		if name, ok := fields["name"]; ok {
			if err := validateMCPEditableMetadata("name", name); err != nil {
				return err
			}
		}
		if models, ok := fields["models"]; ok {
			if err := validateMCPChannelModelsForWrite(models); err != nil {
				return err
			}
		}
	case "newapi_channel_models_apply":
		for _, field := range []string{"add_models", "remove_models", "ignore_models"} {
			var names []string
			if err := common.Unmarshal(fields[field], &names); err != nil {
				return err
			}
			for _, name := range names {
				if len(name) > 255 || strings.TrimSpace(name) == "" || strings.Contains(name, ",") || validateMCPModelName(name) != nil {
					return errors.New("invalid channel model name")
				}
			}
		}
	case "newapi_channel_ollama_pull", "newapi_channel_ollama_delete":
		var name string
		if common.Unmarshal(fields["model_name"], &name) != nil || len(name) > 255 || strings.TrimSpace(name) == "" || validateMCPModelName(name) != nil {
			return errors.New("invalid Ollama model name")
		}
	case "newapi_prefill_group_create", "newapi_prefill_group_update":
		for _, field := range []string{"name", "description"} {
			if err := validateMCPEditableMetadata(field, fields[field]); err != nil {
				return err
			}
		}
		var items []string
		if err := common.Unmarshal(fields["items"], &items); err != nil {
			return err
		}
		for _, item := range items {
			if len(item) > 255 || strings.TrimSpace(item) == "" || validateMCPModelName(item) != nil {
				return errors.New("invalid prefill item")
			}
		}
	case "newapi_groups_save":
		var groups []map[string]json.RawMessage
		if err := common.Unmarshal(fields["groups"], &groups); err != nil {
			return err
		}
		for _, group := range groups {
			var code string
			if common.Unmarshal(group["code"], &code) != nil {
				return errors.New("group code is required")
			}
			if _, err := model.NormalizeGroupCode(code); err != nil {
				return errors.New("invalid group code")
			}
			for _, field := range []string{"name", "description", "code"} {
				if value, exists := group[field]; exists {
					if err := validateMCPEditableMetadata(field, value); err != nil {
						return err
					}
				}
			}
		}
	case "newapi_groups_token_migration_preview", "newapi_groups_tokens_migrate":
		var mode string
		if err := common.Unmarshal(fields["target_group_mode"], &mode); err != nil {
			return err
		}
		_, target := fields["target_group_id"]
		if mode == "auto" && target || mode == "explicit" && !target {
			return errors.New("target group ID must be provided only for explicit migration")
		}
	}
	return nil
}
