package service

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/authz"
)

// 账号工具仅声明控制器实际接收的可编辑字段，不从数据库模型生成输入。
func mcpFullAccountTools() []MCPTool {
	tools := []MCPTool{
		{Name: "newapi_user_edit_get", Category: "users", Path: "/api/user/:id", Description: "读取用户编辑字段及当前分组名称，不返回认证凭据。"},
		{Name: "newapi_user_create", Category: "users", Method: http.MethodPost, Path: "/api/user/", Write: true, Description: "创建用户；密码仅作输入，初始分组由宿主设为 default。角色必须低于操作者。"},
		{Name: "newapi_user_update", Category: "users", Method: http.MethodPut, Path: "/api/user/", Write: true, Description: "完整保存用户名、显示名、分组 code 和备注；可选重置密码。先读取编辑详情，不能省略原配置。"},
		{Name: "newapi_user_delete", Category: "users", Method: http.MethodDelete, Path: "/api/user/:id", Write: true, Description: "永久删除低于操作者角色的用户；软删除使用用户管理工具。"},
		{Name: "newapi_user_manage", Category: "users", Method: http.MethodPost, Path: "/api/user/manage", Write: true, Description: "启用、禁用或软删除用户；角色升降和额度变更使用独立授权工具。"},
		{Name: "newapi_user_quota_adjust", Category: "users", Method: http.MethodPost, Path: "/api/user/manage", Write: true, Description: "调整用户额度，action 固定 add_quota，mode 为 add/subtract/override；单位为原始 quota，非金额。"},
		{Name: "newapi_user_role_set", Category: "authz", MinRole: common.RoleRootUser, Method: http.MethodPost, Path: "/api/user/manage", Write: true, Description: "Root 将用户提升为管理员或降级为普通用户；不支持创建自定义角色。"},
		{Name: "newapi_user_permissions_get", Category: "authz", MinRole: common.RoleRootUser, Path: "/api/user/:id", Description: "Root 读取用户有效权限与保存权限时需要保留的用户编辑字段。"},
		{Name: "newapi_user_permissions_set", Category: "authz", MinRole: common.RoleRootUser, Method: http.MethodPut, Path: "/api/user/", Write: true, Description: "Root 保存用户完整权限矩阵；沿用用户更新接口，必须保留 username/display_name/group/remark。"},
		{Name: "newapi_authz_roles", Category: "authz", Path: "/api/authz/catalog", Description: "查询固定角色、资源、动作及基线权限矩阵；并非角色实体 CRUD。"},
		{Name: "newapi_user_oauth_bindings", Category: "users", Path: "/api/user/:id/oauth/bindings", Description: "查询用户自定义 OAuth 绑定及提供商身份。"},
		{Name: "newapi_user_oauth_unbind", Category: "users", Method: http.MethodDelete, Path: "/api/user/:id/oauth/bindings/:provider_id", Write: true, Description: "解绑目标用户的指定自定义 OAuth 提供商，沿用目标角色权限检查。"},
		{Name: "newapi_user_binding_clear", Category: "users", Method: http.MethodDelete, Path: "/api/user/:id/bindings/:binding_type", Write: true, Description: "清除用户指定登录绑定；可能导致该登录方式失效。"},
		{Name: "newapi_user_passkey_reset", Category: "users", Method: http.MethodDelete, Path: "/api/user/:id/reset_passkey", Write: true, Description: "管理员清除目标用户 Passkey；沿用宿主目标角色检查。"},
		{Name: "newapi_user_2fa_disable", Category: "users", Method: http.MethodDelete, Path: "/api/user/:id/2fa", Write: true, Description: "管理员禁用目标用户双重验证；不读取恢复码或认证种子。"},
		{Name: "newapi_users_2fa_stats", Category: "users", Path: "/api/user/2fa/stats", Description: "查询用户双重验证启用统计。"},
		{Name: "newapi_my_token_create", Category: "tokens", Method: http.MethodPost, Path: "/api/token/", Write: true, Description: "创建本人推理令牌，首次返回 data.id，重放保留 receipt.data.id；随后按 ID 调用本人编辑或 Root 本人密钥读取工具。创建响应不返回密钥，无需按名称查找。"},
		{Name: "newapi_my_token_update", Category: "tokens", Method: http.MethodPut, Path: "/api/token/", Write: true, Description: "完整保存本人令牌配置，先读取编辑详情；状态通过独立工具修改。"},
		{Name: "newapi_my_token_edit_get", Category: "tokens", Path: "/api/token/:id", Description: "读取本人令牌完整可编辑配置和分组显示名，排除密钥。"},
		{Name: "newapi_my_token_auto_groups", Category: "tokens", Path: "/api/token/auto-groups", Description: "查询本人令牌可选自动分组及 max_count（0 表示不限制）；groups 为按宿主顺序去重的提交 code，group_names 为对应显示名称。"},
		{Name: "newapi_my_token_status_set", Category: "tokens", Method: http.MethodPut, Path: "/api/token/", Write: true, Description: "仅启停本人令牌，query.status_only 必须为 true。"},
		{Name: "newapi_my_tokens_delete_batch", Category: "tokens", Method: http.MethodPost, Path: "/api/token/batch", Write: true, Description: "按 ID 批量删除本人令牌，最多 100 个。"},
		{Name: "newapi_my_token_prompts_get", Category: "tokens", Path: "/api/token/:id/prompts", Description: "读取本人令牌的提示词安装摘要、启用状态和顺序。"},
		{Name: "newapi_my_token_prompts_set", Category: "tokens", Method: http.MethodPut, Path: "/api/token/:id/prompts", Write: true, Description: "完整覆盖本人令牌提示词绑定，最多 8 条；空数组卸载全部。"},
		{Name: "newapi_my_token_key", Category: "tokens", MinRole: common.RoleRootUser, Method: http.MethodPost, Path: "/api/token/:id/key", Description: "敏感只读：Root 显式读取本人指定推理令牌完整密钥。沿用后台 data.key 原值，新建密钥不含 sk- 前缀，推理客户端使用时追加该前缀。单独授权，不保存幂等回执，继承密钥读取限流和本人归属检查。"},
		{Name: "newapi_ticket_content_get", Category: "tickets", Path: "/api/ticket-admin/:id", Permission: authz.TicketAdminRead, Description: "读取工单及消息正文；使用 before_message_id 向前翻页，含可识别凭据时拒绝整个响应。"},
		{Name: "newapi_ticket_reply", Category: "tickets", Method: http.MethodPost, Path: "/api/ticket-admin/:id/messages", Write: true, Permission: authz.TicketAdminWrite, Description: "向工单发送管理员回复，可能触发已配置通知；回执不保存正文。"},
		{Name: "newapi_tickets_status_batch", Category: "tickets", Method: http.MethodPut, Path: "/api/ticket-admin/batch/status", Write: true, Permission: authz.TicketAdminWrite, Description: "按 ID/版本批量变更工单状态，最多 100 条，支持 archived。"},
		{Name: "newapi_tickets_delete_batch", Category: "tickets", Method: http.MethodPost, Path: "/api/ticket-admin/batch/delete", Write: true, Permission: authz.TicketAdminWrite, Description: "按 ID/版本批量删除工单，最多 100 条。"},
		{Name: "newapi_ticket_auto_reply_get", Category: "tickets", Path: "/api/ticket-admin/automation/auto-reply", Permission: authz.TicketAdminRead, Description: "读取工单自动回复开关和正文。"},
		{Name: "newapi_ticket_auto_reply_set", Category: "tickets", Method: http.MethodPut, Path: "/api/ticket-admin/automation/auto-reply", Write: true, Permission: authz.TicketAdminWrite, Description: "完整保存工单自动回复配置，回执不保存正文。"},
		{Name: "newapi_ticket_auto_archive_get", Category: "tickets", Path: "/api/ticket-admin/automation/auto-archive", Permission: authz.TicketAdminRead, Description: "读取工单自动归档条件。"},
		{Name: "newapi_ticket_auto_archive_set", Category: "tickets", Method: http.MethodPut, Path: "/api/ticket-admin/automation/auto-archive", Write: true, Permission: authz.TicketAdminWrite, Description: "完整保存自动归档配置，inactive_days 为 1 到 3650。"},
		{Name: "newapi_prompts_content_list", Category: "prompts", Path: "/api/prompt-center/admin/", Description: "分页读取提示词正文与完整技能集合。query.p 默认 1，page_size 默认 5、最多 100；data 为 items/total/page/page_size。响应过大时减小每页条数，单条可用精确读取工具。"},
		{Name: "newapi_prompt_content_get", Category: "prompts", Path: "/api/prompt-center/admin/", Description: "按 query.id 精确读取单个提示词正文、技能资源和可编辑配置；独立正文授权，不存在时返回失败，单条仍受响应大小限制。"},
		{Name: "newapi_prompt_archive_import", Category: "prompts", Method: http.MethodPost, Path: "/api/prompt-center/admin/import/archive", UploadField: "file", Description: "上传并解析单个 ZIP 技能包为未发布草稿，内存解析、不执行脚本、不保存模板。body 为 filename/content_base64，单文件最多 6 MiB；检查草稿后另用创建工具保存。只读操作不保存正文回执。"},
		{Name: "newapi_prompt_store_list", Category: "prompts", Path: "/api/prompt-center/admin/store/", Description: "查询内置或在线技能商店，query.source 为 bundled/online；在线查询仅访问宿主固定商店。"},
		{Name: "newapi_prompt_store_import", Category: "prompts", Method: http.MethodPost, Path: "/api/prompt-center/admin/store/:id/import", StringPathIDs: []string{"id"}, Description: "读取商店技能为未发布草稿，不保存模板；id 使用商店返回的 bundled:名称 或 online:名称。"},
		{Name: "newapi_prompt_repository_import", Category: "prompts", Method: http.MethodPost, Path: "/api/prompt-center/admin/import/repository", Description: "匿名读取公开 GitHub 技能为未发布草稿，不保存模板；沿用宿主 HTTPS GitHub 白名单及禁止重定向，不接受私有仓库凭据。"},
		{Name: "newapi_prompt_repository_update", Category: "prompts", Method: http.MethodPost, Path: "/api/prompt-center/admin/:id/update-repository", Write: true, Description: "按版本从已记录的公开 GitHub 来源更新模板；overwrite_local 明确是否覆盖本地改动，回执不保存正文。"},
		{Name: "newapi_prompt_create", Category: "prompts", Method: http.MethodPost, Path: "/api/prompt-center/admin/", Write: true, Description: "创建提示词或技能集合；category_id=0 为未分类，group_ids 使用稳定 ID。"},
		{Name: "newapi_prompt_update", Category: "prompts", Method: http.MethodPut, Path: "/api/prompt-center/admin/:id", Write: true, Description: "按版本完整保存提示词及技能集合；缺失字段会被清空，先读取正文列表。"},
		{Name: "newapi_prompt_delete", Category: "prompts", Method: http.MethodDelete, Path: "/api/prompt-center/admin/:id", Write: true, Description: "按 query.version 删除提示词并卸载相关令牌绑定。"},
		{Name: "newapi_prompt_category_create", Category: "prompts", Method: http.MethodPost, Path: "/api/prompt-center/admin/categories/", Write: true, Description: "创建提示词分类。"},
		{Name: "newapi_prompt_category_update", Category: "prompts", Method: http.MethodPut, Path: "/api/prompt-center/admin/categories/:id", Write: true, Description: "按版本修改提示词分类名称。"},
		{Name: "newapi_prompt_category_delete", Category: "prompts", Method: http.MethodDelete, Path: "/api/prompt-center/admin/categories/:id", Write: true, Description: "按 query.version 删除分类，所属模板移到未分类。"},
		{Name: "newapi_oauth_providers_list", Category: "oauth", MinRole: common.RoleRootUser, Path: "/api/custom-oauth-provider/", Description: "Root 查询 OAuth 配置，排除 client_secret。"},
		{Name: "newapi_oauth_provider_get", Category: "oauth", MinRole: common.RoleRootUser, Path: "/api/custom-oauth-provider/:id", Description: "Root 查询 OAuth 编辑配置，排除 client_secret。"},
		{Name: "newapi_oauth_discovery", Category: "oauth", MinRole: common.RoleRootUser, Method: http.MethodPost, Path: "/api/custom-oauth-provider/discovery", Description: "Root 读取 HTTP/HTTPS OIDC Discovery 文档，body.well_known_url 为完整文档地址；Issuer 通常追加 /.well-known/openid-configuration。仅返回公开 endpoint 和支持能力，不返回 secret，不创建提供商或保存幂等回执。"},
		{Name: "newapi_oauth_provider_create", Category: "oauth", MinRole: common.RoleRootUser, Method: http.MethodPost, Path: "/api/custom-oauth-provider/", Write: true, Description: "Root 创建 OAuth 提供商；client_secret 仅输入，回执只保留 ID。"},
		{Name: "newapi_oauth_provider_update", Category: "oauth", MinRole: common.RoleRootUser, Method: http.MethodPut, Path: "/api/custom-oauth-provider/:id", Write: true, Description: "Root 部分更新 OAuth 提供商；非指针字符串为空时保留原值，client_secret 为空也保留。"},
		{Name: "newapi_oauth_provider_delete", Category: "oauth", MinRole: common.RoleRootUser, Method: http.MethodDelete, Path: "/api/custom-oauth-provider/:id", Write: true, Description: "Root 删除 OAuth 提供商，可能影响依赖该提供商的用户登录。"},
	}
	for i := range tools {
		t := &tools[i]
		t.TypedArguments = true
		if t.MinRole == 0 {
			t.MinRole = common.RoleAdminUser
		}
		if t.Method == "" {
			t.Method = http.MethodGet
		}
		t.InputSchema = mcpToolInputSchema(*t)
		properties := t.InputSchema["properties"].(map[string]any)
		properties["query"] = mcpAccountObject(nil)
		if body := mcpAccountBodySchema(t.Name); body != nil {
			properties["body"] = body
			if !t.Write {
				t.InputSchema["required"] = append(t.InputSchema["required"].([]string), "body")
			}
		}
		switch t.Name {
		case "newapi_prompts_content_list":
			properties["query"] = mcpAccountObject(map[string]any{
				"p":         map[string]any{"type": "string", "pattern": "^([1-9][0-9]{0,3}|10000)$", "maxLength": 5},
				"page_size": map[string]any{"type": "string", "pattern": "^([1-9][0-9]?|100)$", "maxLength": 3},
			})
		case "newapi_prompt_content_get":
			properties["query"] = mcpAccountObject(map[string]any{"id": map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,9}$", "maxLength": 10}}, "id")
			t.InputSchema["required"] = append(t.InputSchema["required"].([]string), "query")
		case "newapi_prompt_store_list":
			properties["query"] = mcpAccountObject(map[string]any{"source": mcpAccountEnum("bundled", "online")})
		case "newapi_prompt_store_import":
			properties["path"] = mcpAccountObject(map[string]any{"id": map[string]any{"type": "string", "maxLength": 128, "pattern": "^(bundled|online):[A-Za-z0-9][A-Za-z0-9_-]*$"}}, "id")
		case "newapi_my_token_status_set":
			properties["query"] = mcpAccountObject(map[string]any{"status_only": mcpAccountEnum("true")}, "status_only")
			t.InputSchema["required"] = append(t.InputSchema["required"].([]string), "query")
		case "newapi_prompt_delete", "newapi_prompt_category_delete":
			properties["query"] = mcpAccountObject(map[string]any{"version": map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,9}$", "maxLength": 10}}, "version")
			t.InputSchema["required"] = append(t.InputSchema["required"].([]string), "query")
		case "newapi_ticket_content_get":
			properties["query"] = mcpAccountObject(map[string]any{"before_message_id": map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,9}$", "maxLength": 10}})
		case "newapi_user_binding_clear":
			properties["path"].(map[string]any)["properties"].(map[string]any)["binding_type"] = mcpAccountEnum("email", "github", "discord", "oidc", "wechat", "telegram", "linuxdo")
		}
		t.Annotations = map[string]bool{"readOnlyHint": !t.Write, "destructiveHint": t.Write, "idempotentHint": true, "openWorldHint": t.Name == "newapi_ticket_reply" || t.Name == "newapi_prompt_store_list" || t.Name == "newapi_prompt_store_import" || t.Name == "newapi_prompt_repository_import" || t.Name == "newapi_prompt_repository_update" || t.Name == "newapi_oauth_discovery"}
	}
	return tools
}

func mcpAccountObject(fields map[string]any, required ...string) map[string]any {
	if fields == nil {
		fields = map[string]any{}
	}
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": fields, "required": required, "additionalProperties": false}
}

func mcpAccountText(max int) map[string]any {
	return map[string]any{"type": "string", "maxLength": max}
}
func mcpAccountInt(min, max int64) map[string]any {
	return map[string]any{"type": "integer", "minimum": min, "maximum": max}
}
func mcpAccountEnum(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}
func mcpAccountArray(item map[string]any, max int) map[string]any {
	return map[string]any{"type": "array", "items": item, "maxItems": max}
}

// 权限矩阵严格来自已注册的资源/动作，不接受任意资源和非布尔权限。
func mcpAccountPermissionsSchema() map[string]any {
	resources := map[string]any{}
	for _, resource := range authz.Catalog() {
		actions := map[string]any{}
		for _, action := range resource.Actions {
			actions[action.Action] = map[string]any{"type": "boolean"}
		}
		resources[resource.Resource] = mcpAccountObject(actions)
	}
	return mcpAccountObject(resources)
}

func mcpAccountBodySchema(name string) map[string]any {
	id := mcpAccountInt(1, 2147483647)
	flag := map[string]any{"type": "boolean"}
	version := mcpAccountInt(1, 2147483647)
	switch name {
	case "newapi_oauth_discovery":
		return mcpAccountObject(map[string]any{
			"well_known_url": map[string]any{"type": "string", "maxLength": 2048, "pattern": "^https?://[^/?#@\\\\\\s]+(/[^?#@\\\\\\s]*)?$"},
		}, "well_known_url")
	case "newapi_prompt_archive_import":
		return mcpUploadSchema()
	case "newapi_prompt_repository_import":
		return mcpAccountObject(map[string]any{"url": map[string]any{"type": "string", "minLength": 1, "maxLength": 2048, "pattern": "^https://github[.]com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(/[^?#\\\\\\s]*)?$"}}, "url")
	case "newapi_prompt_repository_update":
		return mcpAccountObject(map[string]any{"version": version, "overwrite_local": flag}, "version", "overwrite_local")
	case "newapi_user_create":
		return mcpAccountObject(map[string]any{"username": map[string]any{"type": "string", "minLength": 1, "maxLength": 20}, "password": map[string]any{"type": "string", "minLength": 8, "maxLength": 20}, "display_name": mcpAccountText(20), "role": map[string]any{"type": "integer", "enum": []int{common.RoleCommonUser, common.RoleAdminUser}}}, "username", "password", "role")
	case "newapi_user_update", "newapi_user_permissions_set":
		fields := map[string]any{"id": id, "username": map[string]any{"type": "string", "minLength": 1, "maxLength": 20}, "display_name": mcpAccountText(20), "group": map[string]any{"type": "string", "minLength": 1, "maxLength": 64}, "remark": mcpAccountText(255)}
		required := []string{"id", "username", "display_name", "group", "remark"}
		if name == "newapi_user_update" {
			fields["password"] = map[string]any{"type": "string", "minLength": 8, "maxLength": 20}
		} else {
			fields["admin_permissions"] = mcpAccountPermissionsSchema()
			required = append(required, "admin_permissions")
		}
		return mcpAccountObject(fields, required...)
	case "newapi_user_manage":
		return mcpAccountObject(map[string]any{"id": id, "action": mcpAccountEnum("enable", "disable", "delete")}, "id", "action")
	case "newapi_user_role_set":
		return mcpAccountObject(map[string]any{"id": id, "action": mcpAccountEnum("promote", "demote")}, "id", "action")
	case "newapi_user_quota_adjust":
		return mcpAccountObject(map[string]any{"id": id, "action": mcpAccountEnum("add_quota"), "mode": mcpAccountEnum("add", "subtract", "override"), "value": mcpAccountInt(0, 2147483647)}, "id", "action", "mode", "value")
	case "newapi_my_token_create", "newapi_my_token_update":
		fields := map[string]any{"name": mcpAccountText(50), "expired_time": mcpAccountInt(-1, 253402300799), "remain_quota": mcpAccountInt(0, 2147483647), "unlimited_quota": flag, "model_limits_enabled": flag, "model_limits": mcpAccountText(8192), "allow_ips": mcpAccountText(4096), "group": mcpAccountText(4096), "group_mode": mcpAccountEnum("inherit", "explicit", "auto"), "group_ids": mcpAccountArray(id, 100), "group_ratio_limits": mcpAccountText(8192), "cross_group_retry": flag, "auto_groups": mcpAccountArray(mcpAccountText(64), 100), "group_channel_ids": map[string]any{"type": "object", "additionalProperties": id, "maxProperties": 100}}
		required := strings.Fields("name expired_time remain_quota unlimited_quota model_limits_enabled model_limits allow_ips group group_mode group_ids group_ratio_limits cross_group_retry auto_groups group_channel_ids")
		if name == "newapi_my_token_update" {
			fields["id"] = id
			required = append(required, "id")
		}
		return mcpAccountObject(fields, required...)
	case "newapi_my_token_status_set":
		return mcpAccountObject(map[string]any{"id": id, "status": map[string]any{"type": "integer", "enum": []int{1, 2}}}, "id", "status")
	case "newapi_my_tokens_delete_batch":
		ids := mcpAccountArray(id, 100)
		ids["minItems"] = 1
		return mcpAccountObject(map[string]any{"ids": ids}, "ids")
	case "newapi_my_token_prompts_set":
		binding := mcpAccountObject(map[string]any{"template_id": id, "enabled": flag, "sort_order": mcpAccountInt(-2147483648, 2147483647)}, "template_id", "enabled", "sort_order")
		return mcpAccountObject(map[string]any{"bindings": mcpAccountArray(binding, 8)}, "bindings")
	case "newapi_ticket_reply":
		return mcpAccountObject(map[string]any{"content": map[string]any{"type": "string", "minLength": 1, "maxLength": 10000}}, "content")
	case "newapi_tickets_status_batch", "newapi_tickets_delete_batch":
		refs := mcpAccountArray(mcpAccountObject(map[string]any{"id": id, "version": version}, "id", "version"), 100)
		refs["minItems"] = 1
		fields := map[string]any{"tickets": refs}
		required := []string{"tickets"}
		if name == "newapi_tickets_status_batch" {
			fields["status"] = mcpAccountEnum("open", "processing", "resolved", "closed", "archived")
			required = append(required, "status")
		}
		return mcpAccountObject(fields, required...)
	case "newapi_ticket_auto_reply_set":
		return mcpAccountObject(map[string]any{"enabled": flag, "content": mcpAccountText(10000)}, "enabled", "content")
	case "newapi_ticket_auto_archive_set":
		statuses := mcpAccountArray(mcpAccountEnum("open", "processing", "resolved", "closed"), 4)
		statuses["minItems"] = 1
		return mcpAccountObject(map[string]any{"enabled": flag, "inactive_days": mcpAccountInt(1, 3650), "statuses": statuses}, "enabled", "inactive_days", "statuses")
	case "newapi_prompt_create", "newapi_prompt_update":
		resource := mcpAccountObject(map[string]any{"path": mcpAccountText(512), "content": mcpAccountText(1024 * 1024)}, "path", "content")
		skill := mcpAccountObject(map[string]any{"id": mcpAccountText(128), "title": mcpAccountText(80), "description": mcpAccountText(16384), "path": mcpAccountText(512), "content": mcpAccountText(1024 * 1024), "resources": mcpAccountArray(resource, 4096)}, "title", "description", "content")
		fields := map[string]any{"title": map[string]any{"type": "string", "minLength": 1, "maxLength": 80}, "description": mcpAccountText(16384), "content": mcpAccountText(1024 * 1024), "kind": mcpAccountEnum("prompt", "bundle"), "load_mode": mcpAccountEnum("eager", "progressive"), "category_id": mcpAccountInt(0, 2147483647), "skills": mcpAccountArray(skill, 128), "enabled": flag, "sort_order": mcpAccountInt(-2147483648, 2147483647), "all_groups": flag, "group_ids": mcpAccountArray(id, 100)}
		fields["repository_source"] = mcpAccountObject(map[string]any{"url": mcpAccountText(2048), "ref": mcpAccountText(255), "path": mcpAccountText(512), "commit": map[string]any{"type": "string", "pattern": "^[a-f0-9]{40}$"}, "imported_at": mcpAccountInt(1, 253402300799), "content_hash": map[string]any{"type": "string", "pattern": "^[a-f0-9]{64}$"}}, "url", "ref", "path", "commit", "imported_at", "content_hash")
		required := strings.Fields("title description content kind load_mode category_id skills enabled sort_order all_groups group_ids")
		if name == "newapi_prompt_update" {
			fields["version"] = version
			required = append(required, "version")
		}
		return mcpAccountObject(fields, required...)
	case "newapi_prompt_category_create", "newapi_prompt_category_update":
		fields := map[string]any{"name": map[string]any{"type": "string", "minLength": 1, "maxLength": 40}}
		required := []string{"name"}
		if name == "newapi_prompt_category_update" {
			fields["version"] = version
			required = append(required, "version")
		}
		return mcpAccountObject(fields, required...)
	case "newapi_oauth_provider_create", "newapi_oauth_provider_update":
		fields := map[string]any{}
		for _, field := range strings.Fields("name slug icon client_id client_secret authorization_endpoint token_endpoint user_info_endpoint scopes user_id_field username_field display_name_field email_field well_known access_policy access_denied_message") {
			fields[field] = mcpAccountText(8192)
		}
		fields["enabled"] = flag
		fields["auth_style"] = mcpAccountInt(0, 2)
		required := []string{}
		if name == "newapi_oauth_provider_create" {
			required = strings.Fields("name slug client_id client_secret authorization_endpoint token_endpoint user_info_endpoint")
			for _, field := range required {
				fields[field].(map[string]any)["minLength"] = 1
			}
		}
		body := mcpAccountObject(fields, required...)
		body["minProperties"] = 1
		return body
	}
	return nil
}
