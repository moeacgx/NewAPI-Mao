package service

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullAccountTestTool(t *testing.T, name string) MCPTool {
	t.Helper()
	for _, tool := range mcpFullAccountTools() {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("找不到账号工具 %s", name)
	return MCPTool{}
}

func TestMCPFullAccountRequestContracts(t *testing.T) {
	for _, tc := range []struct {
		name, arguments, path string
		invalid               bool
	}{
		{"newapi_user_create", `{"request_id":"create-1","body":{"username":"alice","password":"Example123","role":1}}`, "/api/user/", false},
		{"newapi_user_create", `{"request_id":"create-1","body":{"username":"alice","password":"Example123","role":100}}`, "", true},
		{"newapi_user_create", `{"request_id":"create-1","body":{"username":"alice","password":"Example123","role":1,"quota":20}}`, "", true},
		{"newapi_user_update", `{"request_id":"edit-1","body":{"id":5,"username":"alice","display_name":"","group":"legacy-code","remark":""}}`, "/api/user/", false},
		{"newapi_user_update", `{"request_id":"edit-1","body":{"id":5,"username":"alice"}}`, "", true},
		{"newapi_user_update", `{"request_id":"edit-1","body":{"id":5,"username":"alice","display_name":"","group":"legacy-code","remark":"","admin_permissions":{}}}`, "", true},
		{"newapi_user_manage", `{"request_id":"state-1","body":{"id":5,"action":"promote"}}`, "", true},
		{"newapi_user_role_set", `{"request_id":"role-1","body":{"id":5,"action":"promote"}}`, "/api/user/manage", false},
		{"newapi_user_quota_adjust", `{"request_id":"quota-1","body":{"id":5,"action":"add_quota","mode":"override","value":0}}`, "/api/user/manage", false},
		{"newapi_user_quota_adjust", `{"request_id":"quota-1","body":{"id":5,"action":"add_quota","mode":"add","value":2147483648}}`, "", true},
		{"newapi_user_permissions_set", `{"request_id":"perms-1","body":{"id":5,"username":"alice","display_name":"","group":"default","remark":"","admin_permissions":{"channel":{"secret_view":false}}}}`, "/api/user/", false},
		{"newapi_user_permissions_set", `{"request_id":"perms-1","body":{"id":5,"username":"alice","display_name":"","group":"default","remark":"","admin_permissions":{"invented":{"write":true}}}}`, "", true},
		{"newapi_user_binding_clear", `{"request_id":"clear-1","path":{"id":"5","binding_type":"github"}}`, "/api/user/5/bindings/github", false},
		{"newapi_user_binding_clear", `{"request_id":"clear-1","path":{"id":"5","binding_type":"password"}}`, "", true},
		{"newapi_my_token_status_set", `{"request_id":"token-1","query":{"status_only":"true"},"body":{"id":1,"status":2}}`, "/api/token/?status_only=true", false},
		{"newapi_my_token_status_set", `{"request_id":"token-1","body":{"id":1,"status":2}}`, "", true},
		{"newapi_my_tokens_delete_batch", `{"request_id":"batch-1","body":{"ids":[1,2]}}`, "/api/token/batch", false},
		{"newapi_my_tokens_delete_batch", `{"request_id":"batch-1","body":{"ids":[]}}`, "", true},
		{"newapi_my_tokens_delete_batch", `{"request_id":"batch-1","body":{"ids":[0]}}`, "", true},
		{"newapi_my_token_prompts_set", `{"request_id":"bindings-1","path":{"id":"1"},"body":{"bindings":[]}}`, "/api/token/1/prompts", false},
		{"newapi_my_token_key", `{"path":{"id":"1"}}`, "/api/token/1/key", false},
		{"newapi_my_token_key", `{"path":{"id":"1"},"body":{"user_id":9}}`, "", true},
		{"newapi_my_token_key", `{"path":{"id":"0"}}`, "", true},
		{"newapi_my_token_auto_groups", `{}`, "/api/token/auto-groups", false},
		{"newapi_my_token_auto_groups", `{"query":{"user_id":"9"}}`, "", true},
		{"newapi_ticket_content_get", `{"path":{"id":"4"},"query":{"before_message_id":"22"}}`, "/api/ticket-admin/4?before_message_id=22", false},
		{"newapi_ticket_reply", `{"request_id":"reply-1","path":{"id":"4"},"body":{"content":"回复\n下一行"}}`, "/api/ticket-admin/4/messages", false},
		{"newapi_tickets_status_batch", `{"request_id":"archive-1","body":{"tickets":[{"id":4,"version":1}],"status":"archived"}}`, "/api/ticket-admin/batch/status", false},
		{"newapi_tickets_status_batch", `{"request_id":"archive-1","body":{"tickets":[{"id":4,"version":0}],"status":"archived"}}`, "", true},
		{"newapi_ticket_auto_archive_set", `{"request_id":"archive-2","body":{"enabled":true,"inactive_days":3651,"statuses":["closed"]}}`, "", true},
		{"newapi_prompt_delete", `{"request_id":"delete-1","path":{"id":"4"},"query":{"version":"1"}}`, "/api/prompt-center/admin/4?version=1", false},
		{"newapi_prompt_delete", `{"request_id":"delete-1","path":{"id":"4"}}`, "", true},
		{"newapi_oauth_provider_update", `{"request_id":"oauth-1","path":{"id":"4"},"body":{"enabled":false,"client_secret":"replacement-secret","auth_style":0}}`, "/api/custom-oauth-provider/4", false},
		{"newapi_oauth_provider_update", `{"request_id":"oauth-1","path":{"id":"4"},"body":{"unknown":"value"}}`, "", true},
		{"newapi_prompt_store_import", `{"path":{"id":"bundled:writing-guide"}}`, "/api/prompt-center/admin/store/bundled:writing-guide/import", false},
		{"newapi_prompt_store_import", `{"path":{"id":"private:writing-guide"}}`, "", true},
		{"newapi_prompt_repository_import", `{"body":{"url":"https://github.com/owner/repo/tree/main/skills"}}`, "/api/prompt-center/admin/import/repository", false},
		{"newapi_prompt_repository_import", `{"body":{"url":"http://127.0.0.1/private"}}`, "", true},
		{"newapi_prompt_repository_import", `{"body":{"url":"https://github.com/owner/repo?token=private"}}`, "", true},
		{"newapi_prompt_repository_update", `{"request_id":"update-repo-1","path":{"id":"4"},"body":{"version":1,"overwrite_local":false}}`, "/api/prompt-center/admin/4/update-repository", false},
		{"newapi_oauth_discovery", `{"body":{"well_known_url":"https://issuer.example/.well-known/openid-configuration"}}`, "/api/custom-oauth-provider/discovery", false},
		{"newapi_oauth_discovery", `{"body":{}}`, "", true},
		{"newapi_oauth_discovery", `{"body":{"well_known_url":"https://issuer.example/config","client_secret":"private"}}`, "", true},
		{"newapi_oauth_discovery", `{"body":{"well_known_url":"https://user:pass@issuer.example/config"}}`, "", true},
		{"newapi_oauth_discovery", `{"body":{"well_known_url":"https://issuer.example/config?token=private"}}`, "", true},
		{"newapi_prompts_content_list", `{}`, "/api/prompt-center/admin/", false},
		{"newapi_prompts_content_list", `{"query":{"p":"10000","page_size":"100"}}`, "/api/prompt-center/admin/?p=10000&page_size=100", false},
		{"newapi_prompts_content_list", `{"query":{"p":"0"}}`, "", true},
		{"newapi_prompts_content_list", `{"query":{"p":"10001"}}`, "", true},
		{"newapi_prompts_content_list", `{"query":{"page_size":"101"}}`, "", true},
		{"newapi_prompts_content_list", `{"query":{"id":"1"}}`, "", true},
		{"newapi_prompt_content_get", `{"query":{"id":"2147483647"}}`, "/api/prompt-center/admin/?id=2147483647", false},
		{"newapi_prompt_content_get", `{"query":{"id":"2147483648"}}`, "", true},
		{"newapi_prompt_content_get", `{"query":{"id":"0"}}`, "", true},
		{"newapi_prompt_content_get", `{}`, "", true},
		{"newapi_prompt_content_get", `{"query":{"id":"1","page_size":"1"}}`, "", true},
	} {
		t.Run(tc.name+"/"+tc.arguments, func(t *testing.T) {
			_, path, _, err := PrepareMCPRequest(fullAccountTestTool(t, tc.name), []byte(tc.arguments))
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.path, path)
		})
	}
}

func TestMCPFullAccountPromptZeroCategoryAndExplicitTokenGroup(t *testing.T) {
	prompt := map[string]any{"title": "测试", "description": "", "content": "正文", "kind": "prompt", "load_mode": "eager", "category_id": 0, "skills": []any{}, "enabled": false, "sort_order": 0, "all_groups": true, "group_ids": []int{}, "version": 1}
	arguments := map[string]any{"request_id": "prompt-1", "path": map[string]string{"id": "2"}, "body": prompt}
	raw, err := common.Marshal(arguments)
	require.NoError(t, err)
	_, _, body, err := PrepareMCPRequest(fullAccountTestTool(t, "newapi_prompt_update"), raw)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"category_id":0`)
	assert.Contains(t, string(body), `"enabled":false`)
	prompt["category_id"] = -1
	raw, err = common.Marshal(arguments)
	require.NoError(t, err)
	_, _, _, err = PrepareMCPRequest(fullAccountTestTool(t, "newapi_prompt_update"), raw)
	require.Error(t, err)
	token := map[string]any{"name": "测试", "expired_time": -1, "remain_quota": 0, "unlimited_quota": false, "model_limits_enabled": false, "model_limits": "", "allow_ips": "", "group": "legacy-code", "group_mode": "explicit", "group_ids": []int{5}, "group_ratio_limits": "", "cross_group_retry": false, "auto_groups": []string{}, "group_channel_ids": map[string]int{}}
	raw, err = common.Marshal(map[string]any{"request_id": "token-2", "body": token})
	require.NoError(t, err)
	_, _, _, err = PrepareMCPRequest(fullAccountTestTool(t, "newapi_my_token_create"), raw)
	require.NoError(t, err)
	token["user_id"] = 99
	raw, err = common.Marshal(map[string]any{"request_id": "token-2", "body": token})
	require.NoError(t, err)
	_, _, _, err = PrepareMCPRequest(fullAccountTestTool(t, "newapi_my_token_create"), raw)
	require.Error(t, err)
}

func TestMCPFullAccountResponseAllowlist(t *testing.T) {
	for _, tc := range []struct{ name, raw, want string }{
		{"newapi_ticket_content_get", `{"success":true,"message":"secret","data":{"ticket":{"id":4,"title":"工单","content":"未知位置"},"messages":[{"id":1,"content":"正文\n下一行","key":"secret"}],"has_more":true,"unknown":"secret"}}`, `{"success":true,"data":{"ticket":{"id":4,"title":"工单"},"messages":[{"id":1,"content":"正文\n下一行"}],"has_more":true}}`},
		{"newapi_ticket_reply", `{"success":true,"data":{"id":1,"ticket_id":4,"content":"sk-sensitive-secret","author_id":3}}`, `{"success":true,"data":{"id":1,"ticket_id":4}}`},
		{"newapi_prompt_create", `{"success":true,"data":{"id":8,"version":1,"content":"client_secret=private","repository_source":{"token":"secret"}}}`, `{"success":true,"data":{"id":8,"version":1}}`},
		{"newapi_oauth_provider_create", `{"success":true,"data":{"id":9,"client_secret":"secret","client_id":"sensitive-input","name":"secret"}}`, `{"success":true,"data":{"id":9}}`},
		{"newapi_oauth_provider_get", `{"success":true,"data":{"id":9,"client_id":"public-client","client_secret":"secret","token_endpoint":"https://issuer.example/token","authorization_endpoint":"https://issuer.example/authorize","enabled":false,"unknown":{"client_secret":"secret"}}}`, `{"success":true,"data":{"id":9,"client_id":"public-client","token_endpoint":"https://issuer.example/token","authorization_endpoint":"https://issuer.example/authorize","enabled":false}}`},
		{"newapi_user_permissions_get", `{"success":true,"data":{"id":5,"group":"legacy","group_name":"显示名","password":"secret","admin_permissions":{"channel":{"read":true,"secret_view":false,"unknown":true},"unknown":{"read":true}}}}`, `{"success":true,"data":{"id":5,"group":"legacy","group_name":"显示名","admin_permissions":{"channel":{"read":true,"secret_view":false}}}}`},
		{"newapi_my_token_edit_get", `{"success":true,"data":{"id":5,"group_details":[{"id":1,"code":"legacy","name":"显示名","exclusive":false}],"key":"sk-real-token","model_limits":"model-a,model-b","allow_ips":null,"auto_groups":null}}`, `{"success":true,"data":{"id":5,"group_details":[{"id":1,"code":"legacy","name":"显示名","exclusive":false}],"model_limits":"model-a,model-b","allow_ips":"","auto_groups":[],"group_ids":[],"group_channel_ids":{}}}`},
		{"newapi_my_token_key", `{"success":true,"data":{"key":"sk-example-secret","client_secret":"hidden"},"key":"sk-other"}`, `{"success":true,"data":{"key":"sk-example-secret"}}`},
		{"newapi_my_token_key", `{"success":true,"data":{"key":"owner1234token5678","client_secret":"hidden"}}`, `{"success":true,"data":{"key":"owner1234token5678"}}`},
		{"newapi_my_tokens_delete_batch", `{"success":true,"data":2}`, `{"success":true,"data":2}`},
		{"newapi_my_token_auto_groups", `{"success":true,"data":{"groups":[],"max_count":0,"key":"private","unknown":{"secret":"private"}}}`, `{"success":true,"data":{"groups":[],"group_names":[],"max_count":0}}`},
		{"newapi_my_token_prompts_set", `{"success":true,"data":[{"template":{"id":5,"version":1,"title":"sk-sensitive"},"enabled":false,"sort_order":0}]}`, `{"success":true,"data":[{"template":{"id":5,"version":1},"enabled":false,"sort_order":0}]}`},
		{"newapi_prompt_repository_update", `{"success":true,"data":{"template":{"id":5,"version":2,"content":"sk-sensitive"},"updated":true}}`, `{"success":true,"data":{"template":{"id":5,"version":2},"updated":true}}`},
		{"newapi_oauth_discovery", `{"success":true,"data":{"well_known_url":"https://issuer.example/discovery","discovery":{"issuer":"https://issuer.example","token_endpoint":"https://issuer.example/token","jwks_uri":"https://issuer.example/keys","token_endpoint_auth_methods_supported":["client_secret_basic"],"claims_parameter_supported":false,"client_secret":"private","jwks":{"keys":["private"]},"unknown":{"secret":"private"}}}}`, `{"success":true,"data":{"well_known_url":"https://issuer.example/discovery","discovery":{"issuer":"https://issuer.example","token_endpoint":"https://issuer.example/token","jwks_uri":"https://issuer.example/keys","token_endpoint_auth_methods_supported":["client_secret_basic"],"claims_parameter_supported":false}}}`},
		{"newapi_prompts_content_list", `{"success":true,"data":{"items":[{"id":1,"content":"完整正文","skills":[{"content":"技能正文","resources":[{"path":"guide.md","content":"资源正文","secret":"private"}]}],"unknown":"private"}],"page":1,"page_size":5,"total":1,"unknown":"private"}}`, `{"success":true,"data":{"items":[{"id":1,"content":"完整正文","skills":[{"content":"技能正文","resources":[{"path":"guide.md","content":"资源正文"}]}]}],"page":1,"page_size":5,"total":1}}`},
		{"newapi_prompt_content_get", `{"success":true,"data":{"id":1,"content":"完整正文","unknown":{"secret":"private"}}}`, `{"success":true,"data":{"id":1,"content":"完整正文"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, handled, err := projectMCPFullAccountResponse(fullAccountTestTool(t, tc.name), []byte(tc.raw))
			require.NoError(t, err)
			require.True(t, handled)
			assert.JSONEq(t, tc.want, string(out))
		})
	}
}

func TestMCPFullAccountArchiveDraftMultipart(t *testing.T) {
	// 用实际 ZIP 和宿主解析器验证上传协议，不访问网络或业务数据库。
	var archive bytes.Buffer
	zipWriter := zip.NewWriter(&archive)
	file, err := zipWriter.Create("guide.md")
	require.NoError(t, err)
	_, err = file.Write([]byte("# 使用指南\n\n保留这段正文。"))
	require.NoError(t, err)
	require.NoError(t, zipWriter.Close())
	tool := fullAccountTestTool(t, "newapi_prompt_archive_import")
	input := map[string]any{"body": map[string]string{"filename": "guide.zip", "content_base64": base64.StdEncoding.EncodeToString(archive.Bytes())}}
	raw, err := common.Marshal(input)
	require.NoError(t, err)
	_, path, body, err := PrepareMCPRequest(tool, raw)
	require.NoError(t, err)
	assert.Equal(t, "/api/prompt-center/admin/import/archive", path)
	assert.False(t, tool.Write, "解析草稿不保存模板，也不保存正文回执")
	encoded, contentType, err := mcpUploadBody(tool.UploadField, body)
	require.NoError(t, err)
	_, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	reader := multipart.NewReader(bytes.NewReader(encoded), params["boundary"])
	part, err := reader.NextPart()
	require.NoError(t, err)
	assert.Equal(t, "file", part.FormName())
	assert.Equal(t, "guide.zip", part.FileName())
	data, err := io.ReadAll(part)
	require.NoError(t, err)
	assert.Equal(t, archive.Bytes(), data)
	draft, err := ParseSkillArchive(data, part.FileName())
	require.NoError(t, err)
	response, err := common.Marshal(map[string]any{"success": true, "data": draft})
	require.NoError(t, err)
	projected, handled, err := projectMCPFullAccountResponse(tool, response)
	require.NoError(t, err)
	require.True(t, handled)
	assert.Contains(t, string(projected), "保留这段正文")
	_, err = reader.NextPart()
	require.ErrorIs(t, err, io.EOF)
	input["body"] = map[string]string{"filename": "../guide.zip", "content_base64": "AAAA"}
	raw, err = common.Marshal(input)
	require.NoError(t, err)
	_, _, _, err = PrepareMCPRequest(tool, raw)
	require.Error(t, err)
}

func TestMCPFullAccountUserQuotaRejectsInvalidInt64(t *testing.T) {
	for _, name := range []string{"newapi_user_edit_get", "newapi_user_permissions_get"} {
		tool := fullAccountTestTool(t, name)
		for _, quota := range []string{"9223372036854775808", "-9223372036854775809", `"1"`, "1.5", "1e3", "null"} {
			t.Run(name+"/"+quota, func(t *testing.T) {
				raw := []byte(`{"success":true,"data":{"id":1,"quota":` + quota + `}}`)
				out, handled, err := projectMCPFullAccountResponse(tool, raw)
				assert.True(t, handled)
				require.Error(t, err)
				assert.Empty(t, out)
			})
		}
	}
}

func TestMCPFullAccountProjectionRejectsUnsafeContent(t *testing.T) {
	tool := fullAccountTestTool(t, "newapi_ticket_content_get")
	for _, content := range []string{"Bearer private-value", "api_key=private", "https://user:pass@example.org/path", "https://example.org/path?token=private", strings.Repeat("x", 10001)} {
		raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"messages": []any{map[string]any{"content": content}}}})
		require.NoError(t, err)
		out, handled, err := projectMCPFullAccountResponse(tool, raw)
		require.True(t, handled)
		require.Error(t, err)
		assert.Empty(t, out)
	}
	_, handled, err := projectMCPFullAccountResponse(tool, []byte(`{"success":false,"message":"secret"}`))
	require.True(t, handled)
	require.Error(t, err)
	_, handled, err = projectMCPFullAccountResponse(MCPTool{Name: "newapi_ticket_get"}, []byte(`{"success":true}`))
	require.NoError(t, err)
	assert.False(t, handled)
	secret := fullAccountTestTool(t, "newapi_my_token_key")
	secret.Write = true
	_, _, err = projectMCPFullAccountResponse(secret, []byte(`{"success":true,"data":{"key":"sk-example-secret"}}`))
	require.Error(t, err)
}

func TestMCPFullAccountAuthorizationAndReceiptBoundary(t *testing.T) {
	for _, tool := range mcpFullAccountTools() {
		require.NotNil(t, mcpAccountResponseSchema(tool.Name), tool.Name)
		if tool.Category == "oauth" || tool.Category == "authz" && tool.Name != "newapi_authz_roles" {
			assert.Equal(t, common.RoleRootUser, tool.MinRole, tool.Name)
		}
		if tool.Category == "tickets" {
			want := authz.TicketAdminRead
			if tool.Write {
				want = authz.TicketAdminWrite
			}
			assert.Equal(t, want, tool.Permission)
		}
		if tool.Write {
			_, _, _, err := PrepareMCPRequest(tool, []byte(`{}`))
			require.Error(t, err, tool.Name)
		}
	}
	key := fullAccountTestTool(t, "newapi_my_token_key")
	assert.Equal(t, http.MethodPost, key.Method)
	assert.False(t, key.Write)
	assert.Equal(t, common.RoleRootUser, key.MinRole)
	assert.True(t, key.Annotations["readOnlyHint"])
}

func TestMCPFullAccountDiscoveryAndContentRemainSeparatelyAuthorized(t *testing.T) {
	user := &model.UserBase{Id: 1, Role: common.RoleRootUser}
	oldKey := &model.MCPKey{Tools: []string{"newapi_prompts_list", "newapi_oauth_provider_get"}}
	visible := MCPVisibleTools(user, oldKey)
	require.Len(t, visible, 2)
	for _, tool := range visible {
		assert.Contains(t, oldKey.Tools, tool.Name)
	}
	discovery := fullAccountTestTool(t, "newapi_oauth_discovery")
	assert.Equal(t, http.MethodPost, discovery.Method)
	assert.False(t, discovery.Write, "Discovery 仅查询公开文档，不产生写回执")
	assert.True(t, discovery.Annotations["readOnlyHint"])
	assert.True(t, discovery.Annotations["openWorldHint"])
	admin := &model.UserBase{Id: 2, Role: common.RoleAdminUser}
	assert.Empty(t, MCPVisibleTools(admin, &model.MCPKey{Tools: []string{discovery.Name}}))
	for _, tool := range []MCPTool{discovery, fullAccountTestTool(t, "newapi_prompt_content_get")} {
		raw := []byte(`{"success":false,"message":"api_key=private","data":{"content":"private"}}`)
		out, success, err := ProjectMCPResponse(tool, raw, http.StatusBadRequest)
		require.NoError(t, err)
		assert.False(t, success)
		assert.NotContains(t, string(out), "private")
	}
	for _, endpoint := range []string{"https://user:pass@issuer.example/token", "https://issuer.example/token?access_token=private"} {
		raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"discovery": map[string]string{"token_endpoint": endpoint}}})
		require.NoError(t, err)
		out, _, err := projectMCPFullAccountResponse(discovery, raw)
		require.Error(t, err)
		assert.Empty(t, out, "公开配置字段不能夹带凭据 URL")
	}
}
