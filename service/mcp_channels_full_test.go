package service

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mcpFullChannelTestTool(t *testing.T, name string) MCPTool {
	t.Helper()
	for _, tool := range mcpFullChannelTools() {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("工具不存在: %s", name)
	return MCPTool{}
}

func TestMCPFullChannelRequestContracts(t *testing.T) {
	for _, test := range []struct {
		name, args, method, path, body string
	}{
		{"newapi_channels_create_batch", `{"body":{"mode":"batch","channel":{"name":"批量渠道","type":1,"key":"key-one\nkey-two","models":"gpt-4o","group_ids":[2]}},"request_id":"batch-create"}`, http.MethodPost, "/api/channel/", `{"mode":"batch","channel":{"name":"批量渠道","type":1,"key":"key-one\nkey-two","models":"gpt-4o","group_ids":[2]}}`},
		{"newapi_channel_config_update", `{"body":{"id":4,"vendor_id":null},"request_id":"vendor-clear"}`, http.MethodPut, "/api/channel/", `{"id":4,"vendor_id":null}`},
		{"newapi_channel_config_update", `{"body":{"id":4,"vendor_id":0,"weight":0,"key":"input-secret","key_mode":"append","models":"gpt-4o,SparkDesk-v4.0"},"request_id":"config-1"}`, http.MethodPut, "/api/channel/", `{"id":4,"vendor_id":0,"weight":0,"key":"input-secret","key_mode":"append","models":"gpt-4o,SparkDesk-v4.0"}`},
		{"newapi_channels_test_all", `{"request_id":"test-1"}`, http.MethodGet, "/api/channel/test", ""},
		{"newapi_channels_balance_refresh", `{"request_id":"balance-1"}`, http.MethodGet, "/api/channel/update_balance", ""},
		{"newapi_channels_status_set", `{"body":{"ids":[2,3],"status":2},"request_id":"status-1"}`, http.MethodPost, "/api/channel/status/batch", `{"ids":[2,3],"status":2}`},
		{"newapi_channel_copy", `{"path":{"id":"4"},"body":{},"query":{"suffix":"_副本","reset_balance":"false"},"request_id":"copy-1"}`, http.MethodPost, "/api/channel/copy/4?reset_balance=false&suffix=_%E5%89%AF%E6%9C%AC", `{}`},
		{"newapi_channel_keys_status", `{"body":{"channel_id":4,"action":"get_key_status","page":1,"page_size":100,"status":3}}`, http.MethodPost, "/api/channel/multi_key/manage", `{"channel_id":4,"action":"get_key_status","page":1,"page_size":100,"status":3}`},
		{"newapi_channel_key_disable", `{"body":{"channel_id":4,"action":"disable_key","key_index":0},"request_id":"key-1"}`, http.MethodPost, "/api/channel/multi_key/manage", `{"channel_id":4,"action":"disable_key","key_index":0}`},
		{"newapi_channel_ollama_delete", `{"body":{"channel_id":4,"model_name":"llama3:8b"},"request_id":"ollama-1"}`, http.MethodDelete, "/api/channel/ollama/delete", `{"channel_id":4,"model_name":"llama3:8b"}`},
		{"newapi_channel_models_fetch", `{"path":{"id":"4"},"request_id":"fetch-1"}`, http.MethodGet, "/api/channel/fetch_models/4", ""},
		{"newapi_channel_models_fetch_preview", `{"body":{"type":1,"key":"input-secret","base_url":"https://example.invalid"},"request_id":"fetch-2"}`, http.MethodPost, "/api/channel/fetch_models", `{"type":1,"key":"input-secret","base_url":"https://example.invalid"}`},
		{"newapi_channel_models_apply", `{"body":{"id":4,"add_models":["SparkDesk-v4.0"],"remove_models":[],"ignore_models":[]},"request_id":"models-1"}`, http.MethodPost, "/api/channel/upstream_updates/apply", `{"id":4,"add_models":["SparkDesk-v4.0"],"remove_models":[],"ignore_models":[]}`},
		{"newapi_groups_code_migration_preview", `{"body":{}}`, http.MethodPost, "/api/group/code-migration/preview", `{}`},
		{"newapi_groups_code_migrate", `{"body":{"confirm":true},"request_id":"migrate-1"}`, http.MethodPost, "/api/group/code-migration", `{"confirm":true}`},
		{"newapi_groups_token_migration_preview", `{"body":{"source_group_id":4,"target_group_mode":"auto"}}`, http.MethodPost, "/api/group/token-migration/preview", `{"source_group_id":4,"target_group_mode":"auto"}`},
		{"newapi_groups_tokens_migrate", `{"body":{"source_group_id":4,"target_group_mode":"explicit","target_group_id":8},"request_id":"migrate-2"}`, http.MethodPost, "/api/group/token-migration", `{"source_group_id":4,"target_group_mode":"explicit","target_group_id":8}`},
		{"newapi_prefill_group_create", `{"body":{"name":"测试模型","type":"model","items":["SparkDesk-v4.0"],"description":""},"request_id":"prefill-1"}`, http.MethodPost, "/api/prefill_group/", `{"name":"测试模型","type":"model","items":["SparkDesk-v4.0"],"description":""}`},
		{"newapi_models_sync", `{"body":{"locale":"ja","overwrite":[{"model_name":"gpt-4o","fields":["vendor","status"]}]},"request_id":"sync-1"}`, http.MethodPost, "/api/models/sync_upstream", `{"locale":"ja","overwrite":[{"model_name":"gpt-4o","fields":["vendor","status"]}]}`},
		{"newapi_channel_codex_refresh", `{"path":{"id":"4"},"body":{},"request_id":"codex-1"}`, http.MethodPost, "/api/channel/4/codex/refresh", `{}`},
		{"newapi_channel_codex_usage", `{"path":{"id":"4"},"request_id":"codex-2"}`, http.MethodGet, "/api/channel/4/codex/usage", ""},
		{"newapi_channel_codex_reset_credits", `{"path":{"id":"4"},"request_id":"codex-3"}`, http.MethodGet, "/api/channel/4/codex/usage/reset-credits", ""},
		{"newapi_channel_codex_usage_reset", `{"path":{"id":"4"},"body":{},"request_id":"codex-4"}`, http.MethodPost, "/api/channel/4/codex/usage/reset", `{}`},
		{"newapi_vendors_search", `{"query":{"keyword":"vendor","p":"2","page_size":"20"}}`, http.MethodGet, "/api/vendors/search?keyword=vendor&p=2&page_size=20", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			tool := mcpFullChannelTestTool(t, test.name)
			args, path, body, err := PrepareMCPRequest(tool, []byte(test.args))
			require.NoError(t, err)
			assert.Equal(t, test.method, tool.Method)
			assert.Equal(t, test.path, path)
			if test.body == "" {
				assert.Empty(t, body)
			} else {
				assert.JSONEq(t, test.body, string(body))
			}
			if tool.Write {
				assert.NotEmpty(t, args.RequestID)
			}
		})
	}
}

func TestMCPFullChannelRejectsUnsafeArguments(t *testing.T) {
	for _, test := range []struct{ name, args string }{
		{"newapi_channels_create_batch", `{"body":{"mode":"single","channel":{}},"request_id":"x"}`},
		{"newapi_channels_create_batch", `{"body":{"mode":"batch","batch_add_set_key_prefix_2_name":true,"channel":{"name":"批量","type":1,"key":"one\ntwo","models":"gpt-4o","group_ids":[2]}},"request_id":"x"}`},
		{"newapi_channels_create_batch", `{"body":{"mode":"batch","channel":{"id":1,"name":"批量","type":1,"key":"one\ntwo","models":"gpt-4o","group_ids":[2]}},"request_id":"x"}`},
		{"newapi_channels_create_batch", `{"body":{"mode":"batch","channel":{"name":"批量","type":1,"key":" \n ","models":"gpt-4o","group_ids":[2]}},"request_id":"x"}`},
		{"newapi_channels_create_batch", `{"body":{"mode":"batch","channel":{"name":"批量","type":1,"key":"one\ntwo","models":"a,,b","group_ids":[2]}},"request_id":"x"}`},
		{"newapi_channel_config_update", `{"body":{"id":4,"vendor_id":-1},"request_id":"x"}`},
		{"newapi_channel_config_update", `{"body":{"id":4,"status":1},"request_id":"x"}`},
		{"newapi_channel_config_update", `{"body":{"id":4,"used_quota":0},"request_id":"x"}`},
		{"newapi_channel_config_update", `{"body":{"id":4,"openai_organization":"[redacted]"},"request_id":"x"}`},
		{"newapi_channel_config_update", `{"body":{"id":4,"models":"a,,b"},"request_id":"x"}`},
		{"newapi_channel_config_update", `{"body":{"id":4,"group_ids":[0]},"request_id":"x"}`},
		{"newapi_channel_config_update", `{"body":{"id":0,"name":"不允许零ID"},"request_id":"x"}`},
		{"newapi_channel_tag_update", `{"body":{"tag":"运营","models":"a,"},"request_id":"x"}`},
		{"newapi_channel_models_fetch_preview", `{"body":{"type":2147483647},"request_id":"x"}`},
		{"newapi_channel_keys_status", `{"body":{"channel_id":4,"action":"delete_key","key_index":0}}`},
		{"newapi_channel_keys_status", `{"body":{"channel_id":4,"action":"get_key_status","page_size":101}}`},
		{"newapi_channel_key_delete", `{"body":{"channel_id":4,"action":"delete_key","key_index":-1},"request_id":"x"}`},
		{"newapi_channel_key_enable", `{"body":{"channel_id":4,"action":"enable_key"},"request_id":"x"}`},
		{"newapi_channels_status_set", `{"body":{"ids":[2,3],"status":3},"request_id":"x"}`},
		{"newapi_channel_ollama_delete", `{"request_id":"x"}`},
		{"newapi_channel_models_apply", `{"body":{"id":4,"add_models":["a,b"],"remove_models":[],"ignore_models":[]},"request_id":"x"}`},
		{"newapi_groups_code_migrate", `{"body":{"confirm":false},"request_id":"x"}`},
		{"newapi_groups_tokens_migrate", `{"body":{"source_group_id":4,"target_group_mode":"auto","target_group_id":7},"request_id":"x"}`},
		{"newapi_groups_tokens_migrate", `{"body":{"source_group_id":4,"target_group_mode":"explicit"},"request_id":"x"}`},
		{"newapi_groups_save", `{"body":{"groups":[],"deleted_ids":[0]},"request_id":"x"}`},
		{"newapi_groups_save", `{"body":{"groups":[],"deleted_ids":[],"option_updates":{"SMTPPassword":"oops"}},"request_id":"x"}`},
		{"newapi_prefill_group_update", `{"body":{"id":1,"name":"仅改名"},"request_id":"x"}`},
		{"newapi_models_sync", `{"body":{"overwrite":[{"model_name":"gpt-4o","fields":["key"]}]},"request_id":"x"}`},
	} {
		t.Run(test.name+test.args, func(t *testing.T) {
			tool := mcpFullChannelTestTool(t, test.name)
			_, _, _, err := PrepareMCPRequest(tool, []byte(test.args))
			require.Error(t, err)
		})
	}
}

func TestMCPFullChannelWriteRequestIDAndPermissions(t *testing.T) {
	for _, tool := range mcpFullChannelTools() {
		t.Run(tool.Name, func(t *testing.T) {
			assert.Equal(t, common.RoleAdminUser, tool.MinRole)
			if tool.Write {
				// schema 必填检查保证 GET 运维同样不能省略 request_id。
				required := tool.InputSchema["required"].([]string)
				assert.Contains(t, required, "request_id")
				assert.False(t, tool.Annotations["readOnlyHint"])
			}
			if tool.Category == "channels" {
				assert.Equal(t, "channel", tool.Permission.Resource)
				assert.NotEmpty(t, tool.Permission.Action)
			}
		})
	}
	for _, name := range []string{"newapi_channels_create_batch", "newapi_channel_config_update", "newapi_channel_tag_update", "newapi_channel_key_delete", "newapi_channel_keys_disabled_delete", "newapi_channel_copy", "newapi_channel_ollama_delete"} {
		assert.Equal(t, authz.ChannelSensitiveWrite, mcpFullChannelTestTool(t, name).Permission)
	}
	assert.Equal(t, authz.ChannelOperate, mcpFullChannelTestTool(t, "newapi_channel_keys_status").Permission)
}

func TestMCPFullChannelBatchCreationBoundaries(t *testing.T) {
	for _, test := range []struct {
		name string
		kind int
		keys string
		ok   bool
	}{
		{"普通换行密钥", 1, "one\ntwo\n", true},
		{"批次数量上限", 1, strings.Repeat("key\n", 1000), true},
		{"批次数量超限", 1, strings.Repeat("key\n", 1001), false},
		{"Vertex对象数组", constant.ChannelTypeVertexAi, `[{"private_key":"local-one"},{"private_key":"local-two"}]`, true},
		{"Vertex空数组", constant.ChannelTypeVertexAi, `[]`, false},
		{"Vertex无效元素", constant.ChannelTypeVertexAi, `[null]`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			args, err := common.Marshal(map[string]any{"request_id": "batch-boundary", "body": map[string]any{"mode": "batch", "channel": map[string]any{"name": "批量", "type": test.kind, "key": test.keys, "models": "gpt-4o", "group_ids": []int{1}}}})
			require.NoError(t, err)
			_, _, _, err = PrepareMCPRequest(mcpFullChannelTestTool(t, "newapi_channels_create_batch"), args)
			if test.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
	out, handled, err := projectMCPFullChannelResponse(mcpFullChannelTestTool(t, "newapi_channels_create_batch"), []byte(`{"success":true,"data":{"key":"never-return","unknown":"never-return"}}`))
	require.NoError(t, err)
	require.True(t, handled)
	assert.JSONEq(t, `{"success":true}`, string(out))
}

func TestMCPFullChannelSafeResponseProjection(t *testing.T) {
	for _, test := range []struct{ name, raw, want string }{
		{"newapi_channel_keys_status", `{"success":true,"message":"secret","data":{"keys":[{"index":0,"status":3,"disabled_time":42,"key_preview":"sk-private","reason":"Bearer credential","unknown":"hidden"}],"total":1,"page":1,"page_size":20,"enabled_count":0,"manual_disabled_count":0,"auto_disabled_count":1,"total_pages":1,"unknown":true}}`, `{"success":true,"data":{"keys":[{"index":0,"status":3,"disabled_time":42}],"total":1,"page":1,"page_size":20,"enabled_count":0,"manual_disabled_count":0,"auto_disabled_count":1,"total_pages":1}}`},
		{"newapi_channel_models_fetch", `{"success":true,"data":["SparkDesk-v4.0","gpt-4o"],"key":"secret"}`, `{"success":true,"data":["SparkDesk-v4.0","gpt-4o"]}`},
		{"newapi_channel_codex_refresh", `{"success":true,"data":{"channel_id":4,"channel_name":"渠道","expires_at":"2026-11-15T01:02:03Z","email":"hidden","account_id":"hidden","key":"hidden"}}`, `{"success":true,"data":{"channel_id":4,"channel_name":"渠道","expires_at":"2026-11-15T01:02:03Z"}}`},
		{"newapi_channel_codex_usage", `{"success":true,"upstream_status":200,"data":{"plan_type":"plus","email":"hidden","account_id":"hidden","rate_limit":{"allowed":true,"limit_reached":false,"primary_window":{"used_percent":12.5,"reset_at":1794700000,"token":"hidden"}},"rate_limit_reset_credits":{"available_count":2}}}`, `{"success":true,"upstream_status":200,"data":{"plan_type":"plus","rate_limit":{"allowed":true,"limit_reached":false,"primary_window":{"used_percent":12.5,"reset_at":1794700000}},"rate_limit_reset_credits":{"available_count":2}}}`},
		{"newapi_channel_codex_usage_reset", `{"success":true,"data":{"windows_reset":2,"code":"untrusted","key":"hidden"}}`, `{"success":true,"data":{"windows_reset":2}}`},
		{"newapi_channel_codex_reset_credits", `{"success":true,"data":{"available_count":2,"total_earned_count":4,"credits":[{"id":"private-id","profile_user_id":"private-id","reset_type":"all","status":"available","expires_at":null}]}}`, `{"success":true,"data":{"available_count":2,"total_earned_count":4,"credits":[{"reset_type":"all","status":"available","expires_at":null}]}}`},
		{"newapi_channel_tag_models", `{"success":true,"data":"SparkDesk-v4.0,gpt-4o"}`, `{"success":true,"data":"SparkDesk-v4.0,gpt-4o"}`},
		{"newapi_channel_models_apply", `{"success":true,"data":{"id":4,"added_models":["SparkDesk-v4.0"],"models":"gpt-4o,SparkDesk-v4.0","settings":"secret"}}`, `{"success":true,"data":{"id":4,"added_models":["SparkDesk-v4.0"],"models":"gpt-4o,SparkDesk-v4.0"}}`},
		{"newapi_groups_list", `{"success":true,"data":[{"id":4,"code":"legacy","name":"高级服务","description":"说明","ratio":1.25,"exclusive":true,"single_user_concurrency_limit":0,"auto_enabled":false,"auto_order":0,"new_secret":"hidden"}],"auto_group":{"description":"自动选择","user_selectable":true,"unknown":"hidden"}}`, `{"success":true,"data":[{"id":4,"code":"legacy","name":"高级服务","description":"说明","ratio":1.25,"exclusive":true,"single_user_concurrency_limit":0,"auto_enabled":false,"auto_order":0}],"auto_group":{"description":"自动选择","user_selectable":true}}`},
		{"newapi_groups_tokens_migrate", `{"success":true,"data":{"source_group":{"id":4,"code":"legacy","name":"高级服务"},"target_group":{"id":0,"code":"auto","name":"自动选择"},"migrated_tokens":7,"cache_invalidation_failed":2,"warning":"connection secret","token":"hidden"}}`, `{"success":true,"data":{"source_group":{"id":4,"code":"legacy","name":"高级服务"},"target_group":{"id":0,"code":"auto","name":"自动选择"},"migrated_tokens":7,"cache_invalidation_failed":2}}`},
		{"newapi_models_sync_preview", `{"success":true,"data":{"missing":["SparkDesk-v4.0"],"conflicts":[{"model_name":"gpt-4o","fields":[{"field":"description","local":"Bearer secret","upstream":"secret"}]}],"source":{"locale":"en","models_url":"https://private.invalid?token=hidden"}}}`, `{"success":true,"data":{"missing":["SparkDesk-v4.0"],"conflicts":[{"model_name":"gpt-4o","fields":[{"field":"description"}]}],"source":{"locale":"en"}}}`},
		{"newapi_prefill_groups", `{"success":true,"data":[{"id":1,"name":"模型集合","type":"model","items":["SparkDesk-v4.0"],"description":"说明","secret":"hidden"}]}`, `{"success":true,"data":[{"id":1,"name":"模型集合","type":"model","items":["SparkDesk-v4.0"],"description":"说明"}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			out, handled, err := projectMCPFullChannelResponse(MCPTool{Name: test.name}, []byte(test.raw))
			require.NoError(t, err)
			require.True(t, handled)
			assert.JSONEq(t, test.want, string(out))
		})
	}
}

func TestMCPFullChannelResponseAboveLegacyLimit(t *testing.T) {
	// 上游模型目录超过旧摘要上限时仍须完整返回，避免丢模型后错误覆盖渠道配置。
	names := make([]string, 400)
	for i := range names {
		names[i] = fmt.Sprintf("model-%04d-%s", i, strings.Repeat("x", 100))
	}
	raw, err := common.Marshal(map[string]any{"success": true, "data": names})
	require.NoError(t, err)
	require.Greater(t, len(raw), MCPMaxResponseBytes)
	tool := mcpFullChannelTestTool(t, "newapi_channel_models_fetch")
	out, handled, err := projectMCPFullChannelResponse(tool, raw)
	require.NoError(t, err)
	require.True(t, handled)
	assert.JSONEq(t, string(raw), string(out))
}

func TestMCPFullChannelConfigurationReadModifyWrite(t *testing.T) {
	tool := mcpFullChannelTestTool(t, "newapi_channel_config_get")
	assert.Equal(t, authz.ChannelSensitiveWrite, tool.Permission)
	raw := []byte(`{"success":true,"data":{"id":4,"type":1,"name":"渠道","models":"gpt-4o","group":"legacy","setting":"{\"force_responses\":false,\"proxy\":\"\"}","model_mapping":"{\"local\":\"gpt-4o\"}","param_override":"{\"temperature\":0,\"stream\":false}","settings":"{\"monitor_enabled\":false}","header_override":"{\"X-Custom-Auth\":\"opaque-credential\"}","key":"never-visible","unknown":"never-visible"}}`)
	result, handled, err := projectMCPFullChannelResponse(tool, raw)
	require.NoError(t, err)
	require.True(t, handled)
	assert.NotContains(t, string(result), "opaque-credential")
	assert.NotContains(t, string(result), "never-visible")
	var response struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, common.Unmarshal(result, &response))
	assert.Equal(t, `{"local":"gpt-4o"}`, response.Data["model_mapping"])
	assert.Equal(t, `{"temperature":0,"stream":false}`, response.Data["param_override"])
	assert.Contains(t, response.Data["redacted_fields"], "header_override")
	var setting map[string]any
	require.NoError(t, common.UnmarshalJsonStr(response.Data["setting"].(string), &setting))
	setting["force_responses"] = true
	updatedSetting, err := common.Marshal(setting)
	require.NoError(t, err)
	body := map[string]any{"id": 4, "type": response.Data["type"], "setting": string(updatedSetting), "model_mapping": response.Data["model_mapping"], "param_override": response.Data["param_override"]}
	args, err := common.Marshal(map[string]any{"body": body, "request_id": "config-roundtrip"})
	require.NoError(t, err)
	update := mcpFullChannelTestTool(t, "newapi_channel_config_update")
	_, _, prepared, err := PrepareMCPRequest(update, args)
	require.NoError(t, err)
	assert.Contains(t, string(prepared), `\"force_responses\":true`)
	body["header_override"] = response.Data["header_override"]
	args, err = common.Marshal(map[string]any{"body": body, "request_id": "config-roundtrip-hidden"})
	require.NoError(t, err)
	_, _, _, err = PrepareMCPRequest(update, args)
	require.ErrorContains(t, err, "redacted")
}

func TestMCPFullChannelConfigurationHidesEmbeddedCredentials(t *testing.T) {
	for _, config := range []string{
		`{"proxy":"socks5://user:opaque@localhost:1080"}`,
		`{"proxy":"https://localhost/path?token=opaque"}`,
		`{"nested":{"access_token":"opaque"}}`,
		`{"nested":"{\"private_key\":\"opaque\"}"}`,
	} {
		raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"id": 1, "setting": config}})
		require.NoError(t, err)
		out, handled, err := projectMCPFullChannelResponse(mcpFullChannelTestTool(t, "newapi_channel_config_get"), raw)
		require.NoError(t, err)
		require.True(t, handled)
		assert.NotContains(t, string(out), "opaque")
		assert.Contains(t, string(out), "[redacted]")
	}
}

func TestMCPFullChannelResponseRejectsCredentialAndShapeConfusion(t *testing.T) {
	for _, raw := range []string{
		`{"success":true,"data":["sk-secret123"]}`,
		`{"success":true,"data":[{"id":"gpt-4o","secret":"hidden"}]}`,
		`{"success":true,"data":"Bearer secret"}`,
	} {
		_, handled, err := projectMCPFullChannelResponse(MCPTool{Name: "newapi_channel_models_fetch"}, []byte(raw))
		require.True(t, handled)
		require.Error(t, err)
	}
	_, handled, err := projectMCPFullChannelResponse(MCPTool{Name: "newapi_channel_keys_status"}, []byte(`{"success":true,"data":{"enabled_count":"sk-secret123"}}`))
	require.True(t, handled)
	require.Error(t, err)
	_, handled, err = projectMCPFullChannelResponse(MCPTool{Name: "newapi_users_list"}, []byte(`{}`))
	require.NoError(t, err)
	assert.False(t, handled)
}

func TestMCPFullChannelPriorityRoundTrip(t *testing.T) {
	revision := strings.Repeat("a", 64)
	raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"revision": revision, "items": []any{map[string]any{"id": 4, "priority": 0}}, "changes": []any{map[string]any{"id": 4, "previous": nil, "priority": 0}}}})
	require.NoError(t, err)
	out, handled, err := projectMCPFullChannelResponse(MCPTool{Name: "newapi_channel_priority"}, raw)
	require.NoError(t, err)
	require.True(t, handled)
	assert.JSONEq(t, string(raw), string(out))
	for _, name := range []string{"newapi_channel_priority_reorder", "newapi_channel_priority_restore"} {
		body := map[string]any{"revision": revision, "preview": true}
		if name == "newapi_channel_priority_reorder" {
			body["id"], body["target_id"], body["position"] = 4, 8, "same"
		} else {
			// 恢复 NULL 时省略 priority，previous 为当前值，用于并发检查。
			body["restore"] = []any{map[string]any{"id": 4, "previous": 0}}
		}
		args, err := common.Marshal(map[string]any{"body": body, "request_id": "priority-1"})
		require.NoError(t, err)
		_, path, _, err := PrepareMCPRequest(mcpFullChannelTestTool(t, name), args)
		require.NoError(t, err)
		assert.Equal(t, "/api/channel/priority/reorder", path)
	}
}

func TestMCPFullChannelGroupCreateAndRename(t *testing.T) {
	for _, id := range []int{0, 4} {
		group := map[string]any{"code": "new-premium", "name": "高级服务", "description": "", "ratio": 1.5, "status": 1, "user_selectable": true, "exclusive": false, "single_user_concurrency_limit": 0, "auto_enabled": true, "auto_order": 0}
		if id > 0 {
			group["id"], group["code"] = id, "legacy"
		}
		args, err := common.Marshal(map[string]any{"body": map[string]any{"groups": []any{group}, "deleted_ids": []int{}}, "request_id": "groups-1"})
		require.NoError(t, err)
		_, _, body, err := PrepareMCPRequest(mcpFullChannelTestTool(t, "newapi_groups_save"), args)
		require.NoError(t, err)
		assert.Contains(t, string(body), "高级服务")
		if id > 0 {
			assert.Contains(t, string(body), `"code":"legacy"`)
		}
	}
}

func TestMCPFullChannelGroupCodeRequired(t *testing.T) {
	for _, code := range []any{nil, "", " \t ", "auto", "ALL", "null", "a,b"} {
		group := map[string]any{"name": "高级服务", "description": "", "ratio": 1, "status": 1, "user_selectable": true, "exclusive": false, "single_user_concurrency_limit": 0, "auto_enabled": false, "auto_order": 0}
		if code != nil {
			group["code"] = code
		}
		raw, err := common.Marshal(map[string]any{"request_id": "bad-code", "body": map[string]any{"groups": []any{group}, "deleted_ids": []int{}}})
		require.NoError(t, err)
		_, _, _, err = PrepareMCPRequest(mcpFullChannelTestTool(t, "newapi_groups_save"), raw)
		require.Error(t, err, "code=%v", code)
	}
}
