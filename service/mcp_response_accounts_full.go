package service

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// 仅在核心判定业务成功后调用；失败正文继续由核心固定消息处理。
// 白名单按对象层级限定，正文和配置绝不通过整个 data 原样放行。
func projectMCPFullAccountResponse(tool MCPTool, raw []byte) ([]byte, bool, error) {
	schema := mcpAccountResponseSchema(tool.Name)
	if schema == nil {
		return nil, false, nil
	}
	if len(raw) > mcpResponseLimit(tool) {
		return nil, true, errors.New("account response too large")
	}
	var object map[string]json.RawMessage
	if err := common.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, true, errors.New("invalid account response")
	}
	var success bool
	if common.Unmarshal(object["success"], &success) != nil || !success {
		return nil, true, errors.New("account projection requires success")
	}
	result := map[string]json.RawMessage{"success": json.RawMessage(`true`)}
	if data, exists := object["data"]; exists {
		// 密钥例外只属于此工具，要求只读且 Root；其他任何工具都不复用它。
		if tool.Name == "newapi_my_token_key" {
			if tool.Write || tool.MinRole != common.RoleRootUser {
				return nil, true, errors.New("invalid secret tool policy")
			}
			var value map[string]json.RawMessage
			var key string
			if common.Unmarshal(data, &value) != nil || common.Unmarshal(value["key"], &key) != nil || len(key) < 1 || len(key) > 256 || strings.ContainsAny(key, "\x00\r\n\t ") {
				return nil, true, errors.New("invalid token secret response")
			}
			encoded, err := common.Marshal(map[string]string{"key": key})
			if err != nil {
				return nil, true, err
			}
			result["data"] = encoded
		} else {
			if tool.Name == "newapi_my_token_auto_groups" {
				var choices struct {
					Groups   []string `json:"groups"`
					MaxCount int      `json:"max_count"`
				}
				if common.GetJsonType(data) != "object" || common.Unmarshal(data, &choices) != nil {
					return nil, true, errors.New("invalid token auto group response")
				}
				if choices.Groups == nil {
					choices.Groups = []string{}
				}
				displayNames := make([]string, 0, len(choices.Groups))
				if len(choices.Groups) > 0 {
					// 批量解析当前显示名及历史别名，提交仍使用宿主返回的业务 code。
					names, err := model.GetGroupDisplayNameMap()
					if err != nil {
						return nil, true, err
					}
					for _, code := range choices.Groups {
						displayNames = append(displayNames, model.FormatGroupDisplayNames(code, names))
					}
				}
				encoded, err := common.Marshal(map[string]any{"groups": choices.Groups, "group_names": displayNames, "max_count": choices.MaxCount})
				if err != nil {
					return nil, true, err
				}
				data = encoded
			}
			// schema 标准化后调用共享标量验证器，保持 enum/range 语义一致。
			encoded, err := common.Marshal(schema)
			if err != nil {
				return nil, true, err
			}
			var normalized map[string]any
			if err = common.Unmarshal(encoded, &normalized); err != nil {
				return nil, true, err
			}
			projected, err := projectMCPAccountData(data, normalized, 0)
			if err != nil {
				return nil, true, err
			}
			if tool.Name == "newapi_my_token_edit_get" {
				// 宿主空集合可能为 null 或省略；编辑读取给出可直接用于完整更新的明确空值。
				var editable map[string]json.RawMessage
				if common.Unmarshal(projected, &editable) != nil || editable == nil {
					return nil, true, errors.New("invalid token edit response")
				}
				for field, empty := range map[string]string{"auto_groups": "[]", "group_ids": "[]", "group_channel_ids": "{}", "allow_ips": `""`} {
					value, exists := editable[field]
					if !exists || common.GetJsonType(value) == "null" {
						editable[field] = json.RawMessage(empty)
					}
				}
				projected, err = common.Marshal(editable)
				if err != nil {
					return nil, true, err
				}
			}
			result["data"] = projected
		}
	} else if tool.Name == "newapi_my_token_key" {
		return nil, true, errors.New("token secret missing")
	}
	projected, err := common.Marshal(result)
	return projected, true, err
}

// 投影白名单与请求白名单相互独立；更新响应只留下结构化收据，不保存正文或配置。
func mcpAccountResponseSchema(name string) map[string]any {
	number := mcpAccountInt(0, 9007199254740991)
	flag := map[string]any{"type": "boolean"}
	group := mcpAccountObject(map[string]any{"id": number, "code": mcpAccountText(64), "name": mcpAccountText(256), "exclusive": flag})
	groups := mcpAccountArray(group, 100)
	switch name {
	case "newapi_user_edit_get", "newapi_user_permissions_get":
		fields := map[string]any{"id": number, "username": mcpAccountText(20), "display_name": mcpAccountText(20), "group": mcpAccountText(64), "group_id": number, "group_name": mcpAccountText(256), "remark": mcpAccountText(255), "role": number, "status": number, "quota": number}
		fields["quota"] = map[string]any{"type": "integer", "format": "int64"}
		if name == "newapi_user_permissions_get" {
			fields["admin_permissions"] = mcpAccountPermissionsSchema()
		}
		return mcpAccountObject(fields)
	case "newapi_authz_roles":
		action := mcpAccountObject(map[string]any{"action": mcpAccountText(128), "label_key": mcpAccountText(256), "description_key": mcpAccountText(256)})
		resource := mcpAccountObject(map[string]any{"resource": mcpAccountText(128), "label_key": mcpAccountText(256), "actions": mcpAccountArray(action, 100)})
		role := mcpAccountObject(map[string]any{"key": mcpAccountText(64), "name": mcpAccountText(128), "built_in": flag, "superuser": flag, "grants": mcpAccountPermissionsSchema()})
		return mcpAccountObject(map[string]any{"resources": mcpAccountArray(resource, 100), "roles": mcpAccountArray(role, 100)})
	case "newapi_user_oauth_bindings":
		return mcpAccountArray(mcpAccountObject(map[string]any{"provider_id": number, "provider_name": mcpAccountText(256), "provider_slug": mcpAccountText(256), "provider_icon": mcpAccountText(2048), "provider_user_id": mcpAccountText(512)}), 100)
	case "newapi_users_2fa_stats":
		return mcpAccountObject(map[string]any{"total_users": number, "enabled_users": number, "enabled_rate": mcpAccountText(32)})
	case "newapi_my_token_auto_groups":
		return mcpAccountObject(map[string]any{"groups": mcpAccountArray(mcpAccountText(64), 10000), "group_names": mcpAccountArray(mcpAccountText(256), 10000), "max_count": mcpAccountInt(0, 2147483647)})
	case "newapi_my_token_edit_get":
		fields := mcpAccountBodySchema("newapi_my_token_update")["properties"].(map[string]any)
		fields["group_details"] = groups
		fields["used_quota"] = number
		fields["status"] = number
		fields["created_time"] = number
		fields["accessed_time"] = number
		return mcpAccountObject(fields)
	case "newapi_my_token_prompts_get":
		fields := mcpAccountBodySchema("newapi_prompt_create")["properties"].(map[string]any)
		delete(fields, "content")
		delete(fields, "skills")
		fields["id"] = number
		fields["version"] = number
		fields["category_name"] = mcpAccountText(256)
		fields["group_details"] = groups
		fields["skill_count"] = number
		fields["skills"] = mcpAccountArray(mcpAccountObject(map[string]any{"id": mcpAccountText(128), "title": mcpAccountText(80), "description": mcpAccountText(8192)}), 128)
		return mcpAccountArray(mcpAccountObject(map[string]any{"template": mcpAccountObject(fields), "enabled": flag, "sort_order": mcpAccountInt(-2147483648, 2147483647)}), 8)
	case "newapi_my_token_key":
		return mcpAccountObject(map[string]any{"key": mcpAccountText(256)})
	case "newapi_ticket_content_get":
		ticket := mcpAccountObject(map[string]any{"id": number, "user_id": number, "title": mcpAccountText(120), "category": mcpAccountText(16), "priority": mcpAccountText(16), "status": mcpAccountText(16), "version": number, "created_at": number, "updated_at": number})
		message := mcpAccountObject(map[string]any{"id": number, "ticket_id": number, "author_id": number, "author_role": mcpAccountText(16), "content": mcpAccountText(10000), "created_at": number})
		return mcpAccountObject(map[string]any{"ticket": ticket, "messages": mcpAccountArray(message, 50), "has_more": flag})
	case "newapi_ticket_auto_reply_get":
		return mcpAccountObject(map[string]any{"enabled": flag, "content": mcpAccountText(10000)})
	case "newapi_ticket_auto_archive_get":
		return mcpAccountBodySchema("newapi_ticket_auto_archive_set")
	case "newapi_prompt_store_list":
		return mcpAccountArray(mcpAccountObject(map[string]any{"id": mcpAccountText(128), "title": mcpAccountText(80), "description": mcpAccountText(16384), "kind": mcpAccountText(16), "category": mcpAccountText(40), "version": mcpAccountText(40), "source": mcpAccountEnum("bundled", "online"), "skill_count": number}), 64)
	case "newapi_prompt_repository_update":
		return mcpAccountObject(map[string]any{"template": mcpAccountObject(map[string]any{"id": number, "version": number}), "updated": flag})
	case "newapi_prompts_content_list", "newapi_prompt_content_get", "newapi_prompt_archive_import", "newapi_prompt_store_import", "newapi_prompt_repository_import":
		fields := mcpAccountBodySchema("newapi_prompt_update")["properties"].(map[string]any)
		fields["id"] = number
		fields["version"] = number
		fields["category_name"] = mcpAccountText(256)
		fields["group_details"] = groups
		fields["skill_count"] = number
		fields["repository_source"].(map[string]any)["properties"].(map[string]any)["locally_modified"] = flag
		if name == "newapi_prompts_content_list" {
			return mcpAccountObject(map[string]any{"items": mcpAccountArray(mcpAccountObject(fields), 100), "total": number, "page": mcpAccountInt(1, 10000), "page_size": mcpAccountInt(1, 100)})
		}
		return mcpAccountObject(fields)
	case "newapi_oauth_discovery":
		discovery := map[string]any{}
		// 仅公开 OIDC 配置可见；提供商附带的密钥、令牌或自定义对象均丢弃。
		for _, field := range strings.Fields("issuer authorization_endpoint token_endpoint userinfo_endpoint jwks_uri registration_endpoint end_session_endpoint revocation_endpoint introspection_endpoint device_authorization_endpoint pushed_authorization_request_endpoint check_session_iframe service_documentation op_policy_uri op_tos_uri") {
			discovery[field] = mcpAccountText(2048)
		}
		for _, field := range strings.Fields("scopes_supported response_types_supported response_modes_supported grant_types_supported subject_types_supported id_token_signing_alg_values_supported id_token_encryption_alg_values_supported id_token_encryption_enc_values_supported userinfo_signing_alg_values_supported userinfo_encryption_alg_values_supported userinfo_encryption_enc_values_supported request_object_signing_alg_values_supported request_object_encryption_alg_values_supported request_object_encryption_enc_values_supported token_endpoint_auth_methods_supported token_endpoint_auth_signing_alg_values_supported revocation_endpoint_auth_methods_supported introspection_endpoint_auth_methods_supported code_challenge_methods_supported claim_types_supported claims_supported claims_locales_supported ui_locales_supported acr_values_supported") {
			discovery[field] = mcpAccountArray(mcpAccountText(256), 256)
		}
		for _, field := range strings.Fields("claims_parameter_supported request_parameter_supported request_uri_parameter_supported require_request_uri_registration frontchannel_logout_supported frontchannel_logout_session_supported backchannel_logout_supported backchannel_logout_session_supported require_pushed_authorization_requests") {
			discovery[field] = flag
		}
		return mcpAccountObject(map[string]any{"well_known_url": mcpAccountText(2048), "discovery": mcpAccountObject(discovery)})
	case "newapi_oauth_providers_list", "newapi_oauth_provider_get":
		fields := mcpAccountBodySchema("newapi_oauth_provider_update")["properties"].(map[string]any)
		delete(fields, "client_secret")
		fields["id"] = number
		schema := mcpAccountObject(fields)
		if name == "newapi_oauth_providers_list" {
			return mcpAccountArray(schema, 100)
		}
		return schema
	case "newapi_my_tokens_delete_batch":
		return number
	case "newapi_user_create", "newapi_user_update", "newapi_user_delete", "newapi_user_manage", "newapi_user_quota_adjust", "newapi_user_role_set", "newapi_user_permissions_set", "newapi_user_oauth_unbind", "newapi_user_binding_clear", "newapi_user_passkey_reset", "newapi_user_2fa_disable", "newapi_my_token_create", "newapi_my_token_update", "newapi_my_token_status_set", "newapi_ticket_reply", "newapi_tickets_status_batch", "newapi_tickets_delete_batch", "newapi_ticket_auto_reply_set", "newapi_ticket_auto_archive_set", "newapi_prompt_create", "newapi_prompt_update", "newapi_prompt_delete", "newapi_prompt_category_create", "newapi_prompt_category_update", "newapi_prompt_category_delete", "newapi_oauth_provider_create", "newapi_oauth_provider_update", "newapi_oauth_provider_delete":
		return mcpAccountObject(map[string]any{"id": number, "version": number, "ticket_id": number, "created_at": number, "updated_count": number, "deleted_count": number, "moved_templates": number, "uninstalled_tokens": number})
	case "newapi_my_token_prompts_set":
		// 控制器返回安装列表，收据只保留模板 ID/版本和启用顺序。
		return mcpAccountArray(mcpAccountObject(map[string]any{"template": mcpAccountObject(map[string]any{"id": number, "version": number}), "enabled": flag, "sort_order": mcpAccountInt(-2147483648, 2147483647)}), 8)
	}
	return nil
}

// 仅标量使用验证器，容器递归投影而非要求所有输入必填字段都出现在响应中。
func projectMCPAccountData(raw json.RawMessage, schema map[string]any, depth int) (json.RawMessage, error) {
	if depth > 24 {
		return nil, errors.New("account response nesting too deep")
	}
	if schema["format"] == "int64" && schema["type"] == "integer" {
		if common.GetJsonType(raw) != "number" {
			return nil, errors.New("invalid account int64 response")
		}
		if _, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64); err != nil {
			return nil, errors.New("invalid account int64 response")
		}
		return raw, nil
	}
	if common.GetJsonType(raw) == "null" {
		return raw, nil
	}
	switch schema["type"] {
	case "object":
		var object map[string]json.RawMessage
		if common.GetJsonType(raw) != "object" || common.Unmarshal(raw, &object) != nil {
			return nil, errors.New("invalid account response object")
		}
		out := map[string]json.RawMessage{}
		properties, _ := schema["properties"].(map[string]any)
		for key, value := range object {
			child, _ := properties[key].(map[string]any)
			if child == nil {
				child, _ = schema["additionalProperties"].(map[string]any)
			}
			if child == nil {
				continue
			}
			// 动态键仅用于分组 ID 映射；权限矩阵使用静态资源白名单。
			if properties == nil {
				id, err := strconv.ParseInt(key, 10, 32)
				if err != nil || id <= 0 {
					return nil, errors.New("invalid account map key")
				}
			}
			projected, err := projectMCPAccountData(value, child, depth+1)
			if err != nil {
				return nil, err
			}
			out[key] = projected
		}
		return common.Marshal(out)
	case "array":
		var items []json.RawMessage
		if common.GetJsonType(raw) != "array" || common.Unmarshal(raw, &items) != nil {
			return nil, errors.New("invalid account response array")
		}
		if max, ok := schema["maxItems"].(float64); ok && float64(len(items)) > max {
			return nil, errors.New("account response array too large")
		}
		child, _ := schema["items"].(map[string]any)
		out := make([]json.RawMessage, 0, len(items))
		for _, value := range items {
			projected, err := projectMCPAccountData(value, child, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, projected)
		}
		return common.Marshal(out)
	default:
		var value any
		if err := common.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		if err := validateMCPSchema(schema, value, 0); err != nil {
			return nil, errors.New("invalid account response scalar")
		}
		if text, ok := value.(string); ok {
			if strings.ContainsRune(text, 0) || mcpMetadataCredential.MatchString(text) {
				return nil, errors.New("account response contains sensitive text")
			}
			for _, candidate := range mcpMetadataURL.FindAllString(text, -1) {
				u, err := url.Parse(candidate)
				if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
					return nil, errors.New("account response URL contains credentials or query")
				}
			}
		}
		return raw, nil
	}
}
