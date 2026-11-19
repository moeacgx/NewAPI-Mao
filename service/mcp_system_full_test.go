package service

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPSystemConfigurationContracts(t *testing.T) {
	previous := common.OptionMap
	common.OptionMap = map[string]string{"SiteName": "旧名称", "payment_setting.compliance_confirmed": "false"}
	t.Cleanup(func() { common.OptionMap = previous })
	for _, tc := range []struct {
		name, tool, arguments string
		valid                 bool
	}{
		{"已知设置", "newapi_setting_update", `{"request_id":"set","body":{"key":"SiteName","value":"新名称"}}`, true},
		{"拒绝未知配置", "newapi_setting_update", `{"request_id":"set","body":{"key":"UnknownOption","value":"1"}}`, false},
		{"保留合规门禁", "newapi_setting_update", `{"request_id":"set","body":{"key":"payment_setting.compliance_confirmed","value":"true"}}`, false},
		{"不能回写遮盖标记", "newapi_setting_update", `{"request_id":"set","body":{"key":"SiteName","value":"[redacted]"}}`, false},
		{"删除归档显式确认", "newapi_conversations_clear", `{"request_id":"clear","body":{"confirm":true}}`, true},
		{"拒绝虚假确认", "newapi_conversations_clear", `{"request_id":"clear","body":{"confirm":false}}`, false},
		{"完整配置零值", "newapi_conversation_config_update", `{"request_id":"save","body":{"expected_version":1,"enabled":false,"group_codes":[],"user_ids":[],"max_body_bytes":1024,"retention_days":0,"max_archive_count":100}}`, true},
		{"拒绝缺少完整字段", "newapi_conversation_config_update", `{"request_id":"save","body":{"expected_version":1,"enabled":false}}`, false},
		{"禁止未来清理", "newapi_log_cleanup_start", `{"request_id":"cleanup","body":{},"query":{"target_timestamp":"253402300799"}}`, false},
		{"节点允许点", "newapi_instance_delete", `{"request_id":"node","path":{"node_name":"node.local-1"}}`, true},
		{"拒绝节点路径穿越", "newapi_instance_delete", `{"request_id":"node","path":{"node_name":"../node"}}`, false},
		{"只读预览POST", "newapi_security_delete_preview", `{"body":{"user_id":1}}`, true},
		{"缓存清理需范围", "newapi_channel_affinity_clear", `{"request_id":"clear"}`, false},
		{"缓存全量明确", "newapi_channel_affinity_clear", `{"request_id":"clear","query":{"all":"true"}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var selected MCPTool
			for _, tool := range MCPTools() {
				if tool.Name == tc.tool {
					selected = tool
					break
				}
			}
			require.NotEmpty(t, selected.Name)
			_, _, _, err := PrepareMCPRequest(selected, []byte(tc.arguments))
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestMCPRatioPresetRoundTrip(t *testing.T) {
	var selected MCPTool
	for _, tool := range MCPTools() {
		if tool.Name == "newapi_ratio_fetch" {
			selected = tool
			break
		}
	}
	require.NotEmpty(t, selected.Name)
	for _, tc := range []struct {
		id    int
		valid bool
	}{{-100, true}, {-101, true}, {-1, false}, {-102, false}, {0, true}, {1, true}} {
		raw, err := common.Marshal(map[string]any{
			"request_id": "preset",
			"body": map[string]any{"upstreams": []any{map[string]any{
				"id": tc.id, "name": "价格来源", "base_url": "https://example.com", "endpoint": "/ratios",
			}}},
		})
		require.NoError(t, err)
		_, _, _, err = PrepareMCPRequest(selected, raw)
		assert.Equal(t, tc.valid, err == nil, "id=%d error=%v", tc.id, err)
	}
}

func TestMCPOperationReceiptResourceIdentity(t *testing.T) {
	for _, tc := range []struct {
		tool, raw, expected string
	}{
		{"newapi_extension_upload", `{"success":true,"data":{"module":{"id":"example-module","version":"1.0","description":"private content"}}}`, `"id":"example-module"`},
		{"newapi_extensions_refresh", `{"success":true,"data":{"modules":[{"id":"example-module","version":"1.0","description":"private content"}]}}`, `"id":"example-module"`},
		{"newapi_plugin_upload", `{"success":true,"data":{"key":"example-plugin","version":"1.0","source":"private content","nested":{"key":"private content"}}}`, `"key":"example-plugin"`},
		{"newapi_plugin_upload_put", `{"success":true,"data":{"key":"example-plugin","version":"1.0","source":"private content","nested":{"key":"private content"}}}`, `"key":"example-plugin"`},
	} {
		result, err := mcpManagementReceipt([]byte(tc.raw), "resource", true, false, tc.tool)
		require.NoError(t, err)
		require.Len(t, result.Content, 1)
		assert.Contains(t, result.Content[0].Text, tc.expected)
		assert.NotContains(t, result.Content[0].Text, "private content")
	}
	result, err := mcpManagementReceipt([]byte(`{"data":{"key":"example-secret","id":1}}`), "other", true, false, "newapi_setting_update")
	require.NoError(t, err)
	assert.NotContains(t, result.Content[0].Text, "example-secret")
}

func TestMCPOptionProjectionKeepsEditableDataAndHidesCredentials(t *testing.T) {
	for _, tc := range []struct {
		key, value string
		hidden     bool
	}{
		{"MaxTokenAutoGroups", "0", false},
		{"ModelRatio", `{"SparkDesk-v4.0":1.2}`, false},
		{"SMTPPassword", "private-value", true},
		{"GitHubClientSecret", "private-value", true},
		{"SMTPToken", "private-value", true},
		{"WeChatServerToken", "private-value", true},
		{"BepusdtAuthToken", "private-value", true},
		{"OkpayMerchantToken", "private-value", true},
		{"channel_affinity_setting.rules", `[{"key_sources":[{"key":"X-Session-ID","type":"header"}]}]`, false},
		{"channel_affinity_setting", `{"rules":[{"key_sources":[{"key":"X-Session-ID","type":"header"}]}]}`, false},
		{"NestedConfig", `{"enabled":true,"api_key":"private-value"}`, true},
		{"NestedConfig", `{"url":"https://user:private-value@example.com"}`, true},
		{"NestedConfig", `{"url":"https://example.com?token=private-value"}`, true},
		{"SiteName", "正常名称", false},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			view := MCPProjectOption(tc.key, tc.value)
			assert.Equal(t, tc.hidden, view.Redacted)
			assert.NotContains(t, view.Value, "private-value")
			if !tc.hidden {
				assert.Equal(t, tc.value, view.Value)
			}
		})
	}
}

func TestMCPSystemDiagnosticsPreserveActualDTOFields(t *testing.T) {
	for _, tc := range []struct {
		tool, raw string
		fields    []string
	}{
		{"newapi_model_guard_records", `{"success":true,"data":{"items":[{"requested_model":"request-model","expected_upstream_models":["upstream-model"],"actual_upstream_model":"wrong-model","consecutive_mismatches":3,"failure_threshold":3,"channel_disabled":true,"unknown_secret":"hidden"}]}}`,
			[]string{"requested_model", "expected_upstream_models", "actual_upstream_model", "consecutive_mismatches", "failure_threshold", "channel_disabled"}},
		{"newapi_security_event_get", `{"success":true,"data":{"api_key_id":2,"api_key_name":"业务令牌","source":"prompt_guard","endpoint":"/v1/chat/completions","channel_groups":[{"id":7,"name":"展示名称","code":"business-code"}],"token_groups":[],"prompt_available":true,"unknown_secret":"hidden"}}`,
			[]string{"api_key_id", "api_key_name", "source", "endpoint", "channel_groups", "token_groups", "prompt_available"}},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			projected, success, err := ProjectMCPResponse(MCPTool{Name: tc.tool, TypedArguments: true}, []byte(tc.raw), 200)
			require.NoError(t, err)
			assert.True(t, success)
			for _, field := range tc.fields {
				assert.Contains(t, string(projected), `"`+field+`"`)
			}
			assert.NotContains(t, string(projected), "hidden")
		})
	}
}

func TestMCPConfigurationProjectionPreservesVersionAndDropsUnknownFields(t *testing.T) {
	raw := []byte(`{"success":true,"data":{"enabled":false,"config_version":7,"endpoints":[{"id":"local","name":"本地节点","base_url":"http://127.0.0.1:11434","has_token":true,"token_status":"configured","token":"private-token"}],"unknown_secret":"private-token"}}`)
	projected, success, err := ProjectMCPResponse(MCPTool{Name: "newapi_security_config", TypedArguments: true}, raw, 200)
	require.NoError(t, err)
	require.True(t, success)
	assert.Contains(t, string(projected), `"config_version":7`)
	assert.Contains(t, string(projected), `"has_token":true`)
	assert.Contains(t, string(projected), `"base_url":"http://127.0.0.1:11434"`)
	assert.NotContains(t, string(projected), "private-token")
	preview, _, err := ProjectMCPResponse(MCPTool{Name: "newapi_security_delete_preview", TypedArguments: true}, []byte(`{"success":true,"data":{"matched_count":12,"snapshot_max_id":32,"confirmation_token":"preview-scope","expires_at":1234}}`), 200)
	require.NoError(t, err)
	assert.Contains(t, string(preview), `"matched_count":12`)
	assert.Contains(t, string(preview), `"confirmation_token":"preview-scope"`)
}

func TestMCPFileUploadPreservesBytesAndRejectsPaths(t *testing.T) {
	input := mcpUploadInput{Filename: "plugin.zip", ContentBase64: base64.StdEncoding.EncodeToString([]byte("binary\x00payload"))}
	raw, err := common.Marshal(input)
	require.NoError(t, err)
	body, contentType, err := mcpUploadBody("file", raw)
	require.NoError(t, err)
	_, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	part, err := reader.NextPart()
	require.NoError(t, err)
	assert.Equal(t, "file", part.FormName())
	assert.Equal(t, "plugin.zip", part.FileName())
	content, err := io.ReadAll(part)
	require.NoError(t, err)
	assert.Equal(t, []byte("binary\x00payload"), content)
	_, err = reader.NextPart()
	assert.ErrorIs(t, err, io.EOF)
	for _, filename := range []string{"../plugin.zip", `C:\plugin.zip`, "x\r\nfile.zip"} {
		input.Filename = filename
		raw, err = common.Marshal(input)
		require.NoError(t, err)
		_, _, err = mcpUploadBody("file", raw)
		require.Error(t, err)
	}
}

func TestMCPRatioInputsCannotRedirectChannelCredentials(t *testing.T) {
	for _, tc := range []struct {
		input string
		valid bool
	}{
		{`{"upstreams":[{"name":"本地","base_url":"http://127.0.0.1:3000","endpoint":"/api/pricing"}]}`, true},
		{`{"upstreams":[{"name":"恶意","base_url":"https://example.com","endpoint":"https://other.example/api"}]}`, false},
		{`{"upstreams":[{"name":"恶意","base_url":"https://example.com","endpoint":"//other.example/api"}]}`, false},
		{`{"upstreams":[{"name":"恶意","base_url":"https://example.com","endpoint":"openrouter"}]}`, false},
		{`{"upstreams":[{"name":"恶意","base_url":"https://user:pass@example.com","endpoint":"/api/pricing"}]}`, false},
		{`{"channel_ids":[],"upstreams":[]}`, false},
	} {
		err := validateMCPRatioRequest([]byte(tc.input))
		if tc.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func TestMCPTypedResponseBoundRemainsFinite(t *testing.T) {
	raw := []byte(`{"success":true,"data":{"content":"` + strings.Repeat("x", 40000) + `"}}`)
	_, _, err := ProjectMCPResponse(MCPTool{Name: "newapi_conversation_get", TypedArguments: true}, raw, 200)
	require.NoError(t, err)
	_, _, err = ProjectMCPResponse(MCPTool{Name: "newapi_conversation_get"}, raw, 200)
	require.Error(t, err, "旧摘要工具不应扩大限制")
}

func TestMCPManagementReceiptNeverRetainsEditableContent(t *testing.T) {
	result, err := mcpManagementReceipt([]byte(`{"success":true,"data":{"id":42,"version":2,"name":"名称","description":"Bearer private-credential","html":"api_key=private-credential","settings":{"secret":"private-credential"},"items":[{"id":7,"content":"private-credential"}]}}`), "receipt-1", true, false)
	require.NoError(t, err)
	require.Len(t, result.Content, 1)
	assert.Contains(t, result.Content[0].Text, `"id":42`)
	assert.Contains(t, result.Content[0].Text, `"version":2`)
	assert.Contains(t, result.Content[0].Text, `"id":7`)
	assert.NotContains(t, result.Content[0].Text, "private-credential")
	assert.NotContains(t, result.Content[0].Text, "description")
	assert.NotContains(t, result.Content[0].Text, "settings")
	transient, err := mcpManagementReceipt([]byte(`{"success":true,"data":["generated-secret"]}`), "receipt-2", true, true)
	require.NoError(t, err)
	assert.NotContains(t, transient.Content[0].Text, "generated-secret")
	assert.Contains(t, transient.Content[0].Text, `"result_not_retained":true`)
	assert.False(t, transient.IsError)
}

func TestMCPPricingRejectsNullAndLayeredAliasBypasses(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"ModelRatio", `{"model":null}`},
		{"GroupGroupRatio", `{"group":{"target":null}}`},
		{"GroupGroupRatio", `{"group":null}`},
		{"group_ratio_setting.group_ratio", `{"group":-1}`},
		{"group_ratio_setting.group_ratio", `{"broken"`},
		{"group_ratio_setting.group_group_ratio", `{"group":{"target":false}}`},
	} {
		require.Error(t, validateMCPOptionValue(tc.key, tc.value), tc.key+" "+tc.value)
	}
	require.NoError(t, validateMCPOptionValue("ModelRatio", `{"free-model":0,"paid-model":1.5}`))
	require.NoError(t, validateMCPOptionValue("group_ratio_setting.group_ratio", `{"group":1.5}`))
}
