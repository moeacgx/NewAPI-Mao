package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 本组测试不连接业务数据库和外部服务；包级 TestMain 仅使用进程私有临时 SQLite。
func fullOperationTestTool(t *testing.T, name string) MCPTool {
	t.Helper()
	for _, tool := range mcpFullOperationTools() {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("未找到操作工具 %s", name)
	return MCPTool{}
}

func TestMCPFullOperationsCatalogContracts(t *testing.T) {
	names := map[string]bool{}
	for _, tool := range MCPTools() {
		require.False(t, names[tool.Name], "重复工具：%s", tool.Name)
		names[tool.Name] = true
	}
	for _, tool := range mcpFullOperationTools() {
		t.Run(tool.Name, func(t *testing.T) {
			require.NotNil(t, mcpFullOperationResponseShape(tool.Name), "必须有明确响应合同")
			require.NotEmpty(t, tool.Path)
			require.True(t, tool.TypedArguments)
			if tool.Category == "notifications" || tool.Category == "extensions" && tool.Write {
				assert.Equal(t, common.RoleRootUser, tool.MinRole)
			}
			if tool.Write {
				assert.Contains(t, tool.InputSchema["required"], "request_id")
				assert.False(t, tool.Annotations["readOnlyHint"])
			}
			if tool.Category == "deployments" || tool.Name == "newapi_notification_bot_test" || tool.Name == "newapi_extension_rate_preview" {
				assert.True(t, tool.Annotations["openWorldHint"])
			}
			assert.NotContains(t, tool.Path, "dryrun", "宿主 501 不能作为已支持工具")
		})
	}
}

func TestMCPFullOperationsInputs(t *testing.T) {
	cases := []struct {
		name  string
		input string
		path  string
		valid bool
	}{
		{"newapi_notification_bot_create", `{"request_id":"a","body":{"name":"机器人","token":"input-only","enabled":false}}`, "/api/notification/bots", true},
		{"newapi_notification_bot_update", `{"request_id":"a","path":{"id":"3"},"body":{"enabled":false}}`, "/api/notification/bots/3", true},
		{"newapi_notification_bot_update", `{"request_id":"a","path":{"id":"-1"},"body":{"enabled":false}}`, "", false},
		{"newapi_notification_task_create", `{"request_id":"a","body":{"name":"告警","event_type":"channel_disabled","bot_id":1,"template":"{{channel_name}}","enabled":false,"targets":[{"chat_id":"-1001","id":0,"enabled":false}],"filter_config":{"prefix_dedup_seconds":0}}}`, "/api/notification/tasks", true},
		{"newapi_notification_task_create", `{"request_id":"a","body":{"name":"告警","event_type":"channel_disabled","bot_id":0,"template":"","enabled":true,"targets":[{"chat_id":"1"}]}}`, "", false},
		{"newapi_notification_task_create", `{"request_id":"a","body":{"name":"告警","event_type":"channel_disabled","bot_id":1,"template":"","enabled":true,"targets":[]}}`, "", false},
		{"newapi_notification_task_update", `{"request_id":"a","path":{"id":"1"},"body":{"name":"不能缺完整配置"}}`, "", false},
		{"newapi_extension_enabled_set", `{"request_id":"a","path":{"id":"okx-alipay-rate"},"body":{"enabled":false}}`, "/api/extension-admin/okx-alipay-rate/enabled", true},
		{"newapi_extension_enabled_set", `{"request_id":"a","path":{"id":"../escape"},"body":{"enabled":true}}`, "", false},
		{"newapi_plugin_runtime_set", `{"request_id":"a","body":{"enabled":false,"source":"hidden"}}`, "", false},
		{"newapi_plugin_upload", `{"request_id":"a","body":{"source":"exports.meta={}","remark":"说明"}}`, "/api/plugin/task", true},
		{"newapi_plugin_upload_put", `{"request_id":"a","body":{"source":"exports.meta={}"}}`, "/api/plugin/task", true},
		{"newapi_plugin_source_get", `{"path":{"key":"demo"},"query":{"version":"1.0.0"}}`, "/api/plugin/task/demo?version=1.0.0", true},
		{"newapi_extension_notification_publish", `{"request_id":"a","path":{"id":"orders"},"body":{"event_type":"created","event_key":"order:1","payload":{"name":"测试","amount":0,"urgent":false,"optional":null}}}`, "/api/extensions/orders/notification-events", true},
		{"newapi_extension_notification_publish", `{"request_id":"a","path":{"id":"orders"},"body":{"event_type":"created","event_key":"order:1","payload":{"nested":{"value":1}}}}`, "", false},
		{"newapi_extension_notification_publish", `{"request_id":"a","path":{"id":"orders"},"body":{"event_type":"created","event_key":"order:1","payload":{"array":[1]}}}`, "", false},
		{"newapi_extension_notification_publish", `{"request_id":"a","path":{"id":"orders"},"body":{"event_type":"created","event_key":"","payload":{}}}`, "", false},
		{"newapi_deployment_extend", `{"request_id":"a","path":{"id":"cluster-abc-123"},"body":{"duration_hours":24}}`, "/api/deployments/cluster-abc-123/extend", true},
		{"newapi_deployment_extend", `{"request_id":"a","path":{"id":"cluster-abc-123"},"body":{"duration_hours":-1}}`, "", false},
		{"newapi_deployment_extend", `{"request_id":"a","path":{"id":"cluster-abc-123"},"body":{"duration_hours":2147483647}}`, "", false},
		{"newapi_deployment_container_get", `{"path":{"id":"cluster-1","container_id":"worker-2"}}`, "/api/deployments/cluster-1/containers/worker-2", true},
		{"newapi_deployment_settings_test_connection", `{"request_id":"test-settings","body":{"api_key":"input-only"}}`, "/api/deployments/settings/test-connection", true},
		{"newapi_deployment_logs", `{"path":{"id":"cluster-1"},"query":{"container_id":"worker-2","follow":"false","limit":"100"}}`, "/api/deployments/cluster-1/logs?container_id=worker-2&follow=false&limit=100", true},
		{"newapi_deployment_logs", `{"path":{"id":"cluster-1"}}`, "", false},
		{"newapi_deployment_logs", `{"path":{"id":"cluster-1"},"query":{"container_id":"../worker"}}`, "", false},
		{"newapi_deployment_logs", `{"path":{"id":"cluster-1"},"query":{"container_id":"worker","follow":"true"}}`, "", false},
		{"newapi_deployment_logs", `{"path":{"id":"cluster-1"},"query":{"container_id":"worker","limit":"101"}}`, "", false},
		{"newapi_deployment_logs", `{"path":{"id":"cluster-1"},"query":{"container_id":"worker","limit":"0"}}`, "", false},
		{"newapi_deployment_logs", `{"path":{"id":"cluster-1"},"query":{"container_id":"worker","headers":"hidden"}}`, "", false},
		{"newapi_deployment_logs", `{"path":{"id":"cluster-1"},"query":{"container_id":"worker","start_time":"invalid"}}`, "", false},
		{"newapi_deployment_replicas", `{"query":{"hardware_id":"1","gpu_count":"8"}}`, "/api/deployments/available-replicas?gpu_count=8&hardware_id=1", true},
		{"newapi_deployment_replicas", `{"query":{"hardware_id":"1","gpu_count":"-3"}}`, "", false},
		{"newapi_deployment_name_check", `{}`, "", false},
		{"newapi_deployment_update", `{"request_id":"a","path":{"id":"cluster-1"},"body":{"traffic_port":0,"secret_env_variables":{"API_KEY":"write-only"}}}`, "/api/deployments/cluster-1", true},
		{"newapi_extension_rate_config_set", `{"request_id":"a","body":{"rate_api_url":"","side":"buy","tier":3,"adjustment_type":"absolute","adjustment_value":0}}`, "/api/extension-admin/okx-alipay-rate/config", true},
		{"newapi_extension_rate_config_set", `{"request_id":"a","body":{"rate_api_url":"","side":"invalid","tier":3,"adjustment_type":"absolute","adjustment_value":0}}`, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name+tc.input, func(t *testing.T) {
			_, path, body, err := PrepareMCPRequest(fullOperationTestTool(t, tc.name), []byte(tc.input))
			if !tc.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.path, path)
			if strings.Contains(tc.input, `"enabled":false`) {
				assert.Contains(t, string(body), `"enabled":false`)
			}
		})
	}
}

func TestMCPFullOperationsArrayAndVersionSchema(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"newapi_plugin_marketplace_sources_set", `{"request_id":"clear","body":[]}`},
		{"newapi_plugin_marketplace_sources_set", `{"request_id":"set","body":[{"name":"官方","index_url":"https://example.org/index.json"}]}`},
		{"newapi_plugin_version_delete", `{"request_id":"delete","path":{"key":"demo","version":"1.2.3-beta.1+build.7"}}`},
	} {
		t.Run(tc.name+tc.input, func(t *testing.T) {
			tool := fullOperationTestTool(t, tc.name)
			schemaJSON, err := common.Marshal(tool.InputSchema)
			require.NoError(t, err)
			var schema map[string]any
			var value any
			require.NoError(t, common.Unmarshal(schemaJSON, &schema))
			require.NoError(t, common.Unmarshal([]byte(tc.input), &value))
			require.NoError(t, validateMCPSchema(schema, value, 0))
			_, _, _, err = PrepareMCPRequest(tool, []byte(tc.input))
			require.NoError(t, err, "核心必须保留数组和语义版本路径合同")
		})
	}
}

func TestMCPFullOperationsEditableProjection(t *testing.T) {
	cases := []struct {
		name, data string
		keep, hide []string
	}{
		{"newapi_notification_tasks_edit_list", `[{"id":1,"template":"标题\n{{channel_name}}","event_type":"channel_disabled","targets":[{"id":2,"chat_id":"-1001","mention_name":"管理员","token":"hidden-target"}],"filter_config":{"status_codes":"401,429","error_keywords":["余额不足"],"prefix_dedup_seconds":60},"source":"hidden-source"}]`, []string{"标题", "channel_name", "-1001", "余额不足", "prefix_dedup_seconds"}, []string{"hidden-target", "hidden-source"}},
		{"newapi_notification_bots", `[{"id":1,"token_configured":true,"token":"hidden-token","last_test_error":"hidden-error"}]`, []string{"token_configured", "true"}, []string{"hidden-token", "hidden-error"}},
		{"newapi_notification_event_definitions", `[{"value":"extension.notice","default_template":"{{message}}","variables":["message"],"sample_payload":{"secret":"hidden"}}]`, []string{"extension.notice", "default_template", "variables"}, []string{"hidden", "sample_payload"}},
		{"newapi_notification_preview", `{"html":"<b>标题</b>\n预览","text_length":5,"token":"hidden"}`, []string{"标题", "预览", "text_length"}, []string{"hidden"}},
		{"newapi_plugin_get", `{"key":"demo","source":"hidden-source","version":"1.2.3","marketplace":{"name":"官方","index_url":"https://example.org/index.json","token":"hidden"},"meta":{"key":"demo","models":["SparkDesk-v1"],"description":{"zh":"说明"},"auth":{"type":"bearer","token":"hidden-token"},"routes":[{"method":"POST","path":"/demo","type":"submit"}],"usageSchema":{"input_tokens":{"type":"number","unit":"tokens"}}}}`, []string{"demo", "1.2.3", "SparkDesk-v1", "说明", "usageSchema", "input_tokens", "/demo"}, []string{"hidden-source", "hidden-token", "\"auth\""}},
		{"newapi_plugin_version_delete", `{"DeletedActive":false,"Promoted":null}`, []string{"DeletedActive", "false", "Promoted"}, nil},
		{"newapi_extensions_inventory", `{"root":"hidden-root","modules":[{"id":"okx-alipay-rate","enabled":false,"runtime":{"type":"http","identity_secret":"hidden-secret"},"error":"hidden-path"}]}`, []string{"modules", "okx-alipay-rate", "error_configured"}, []string{"hidden-root", "hidden-secret", "hidden-path"}},
		{"newapi_extension_upload", `{"root":"hidden-root","module":{"id":"test-module","version":"1.0.0","enabled":false}}`, []string{"test-module", "1.0.0", "false"}, []string{"hidden-root"}},
		{"newapi_extension_rate_config", `{"rate_api_url":"https://example.org/rate","side":"buy","tier":3,"adjustment_type":"absolute","adjustment_value":"0"}`, []string{"https://example.org/rate", "adjustment_value"}, nil},
		{"newapi_deployment_get", `{"id":"cluster-1","hardware_id":8,"container_config":{"image_url":"registry.example.org/image:v1","entrypoint":["python","serve.py"],"traffic_port":8080,"env_variables":{"ARBITRARY":"hidden-env"},"registry_secret":"hidden-registry"}}`, []string{"cluster-1", "image:v1", "serve.py", "8080", "env_variables_configured"}, []string{"hidden-env", "hidden-registry"}},
		{"newapi_deployment_container_get", `{"container_id":"worker-1","status":"running","public_url":"https://example.org/?access=hidden-url","events":[{"time":1,"message":"hidden-log"}]}`, []string{"worker-1", "running", "[redacted]"}, []string{"hidden-url", "hidden-log"}},
		{"newapi_deployment_settings", `{"provider":"io.net","configured":true,"can_connect":true,"api_key":"hidden"}`, []string{"configured", "can_connect"}, []string{"hidden"}},
		{"newapi_deployment_price_estimate", `{"estimated_cost":0,"currency":"usdc","estimation_valid":true,"price_breakdown":{"compute_cost":0,"total_cost":0,"hourly_rate":0}}`, []string{"estimated_cost", "hourly_rate", "estimation_valid"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(`{"success":true,"message":"hidden-envelope","data":` + tc.data + `}`)
			got, handled, err := projectMCPFullOperationResponse(MCPTool{Name: tc.name, TypedArguments: true}, raw)
			require.NoError(t, err)
			require.True(t, handled)
			for _, text := range tc.keep {
				assert.Contains(t, string(got), text)
			}
			for _, text := range tc.hide {
				assert.NotContains(t, string(got), text)
			}
			assert.NotContains(t, string(got), "hidden-envelope")
		})
	}
	_, handled, err := projectMCPFullOperationResponse(MCPTool{Name: "unregistered"}, []byte(`{}`))
	require.NoError(t, err)
	assert.False(t, handled)
}

func TestMCPFullOperationsLargeEditableTextAndFailure(t *testing.T) {
	text := strings.Repeat("正文\n", 12000)
	raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"html": text, "text_length": 36000}})
	require.NoError(t, err)
	tool := fullOperationTestTool(t, "newapi_notification_preview")
	got, handled, err := projectMCPFullOperationResponse(tool, raw)
	require.NoError(t, err)
	require.True(t, handled)
	var decoded struct {
		Data struct {
			HTML string `json:"html"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(got, &decoded))
	assert.Equal(t, text, decoded.Data.HTML)
	failed, success, err := ProjectMCPResponse(tool, []byte(`{"success":false,"message":"hidden-error","data":{"html":"hidden-payload"}}`), http.StatusNotImplemented)
	require.NoError(t, err)
	assert.False(t, success)
	assert.Contains(t, string(failed), "501")
	assert.NotContains(t, string(failed), "hidden")
}

func TestMCPFullOperationsMultipartContract(t *testing.T) {
	var archive bytes.Buffer
	zipWriter := zip.NewWriter(&archive)
	entry, err := zipWriter.Create("manifest.json")
	require.NoError(t, err)
	_, err = entry.Write([]byte(`{"id":"test-module"}`))
	require.NoError(t, err)
	require.NoError(t, zipWriter.Close())
	encoded := base64.StdEncoding.EncodeToString(archive.Bytes())
	tool := fullOperationTestTool(t, "newapi_extension_upload")
	input := map[string]string{"filename": "extension.zip", "content_base64": encoded, "archiveSha256": strings.Repeat("a", 64), "expectedId": "test-module", "expectedVersion": "1.0.0", "catalogUrl": "https://raw.githubusercontent.com/moeacgx/maolaonewapi-extensions/main/catalog.json", "archivePath": "published/test-module/1.0.0/test-module-1.0.0.zip"}
	body, err := common.Marshal(input)
	require.NoError(t, err)
	wire, contentType, err := mcpUploadBody(tool.UploadField, body)
	require.NoError(t, err)
	mediaType, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	assert.Equal(t, "multipart/form-data", mediaType)
	reader := multipart.NewReader(bytes.NewReader(wire), params["boundary"])
	file, err := reader.NextPart()
	require.NoError(t, err)
	assert.Equal(t, "file", file.FormName())
	assert.Equal(t, "extension.zip", file.FileName())
	assert.NotEmpty(t, file.Header.Get("Content-Type"))
	actual, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, archive.Bytes(), actual)
	fields := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		value, err := io.ReadAll(part)
		require.NoError(t, err)
		fields[part.FormName()] = string(value)
	}
	assert.Equal(t, map[string]string{"archiveSha256": input["archiveSha256"], "expectedId": input["expectedId"], "expectedVersion": input["expectedVersion"], "catalogUrl": input["catalogUrl"], "archivePath": input["archivePath"]}, fields)
	for _, bad := range []map[string]string{
		{"filename": "extension.zip", "content_base64": "not_base64"},
		{"filename": "../escape.zip", "content_base64": encoded},
		{"filename": "extension.zip", "content_base64": encoded, "expectedId": "partial-marketplace"},
	} {
		badJSON, err := common.Marshal(bad)
		require.NoError(t, err)
		_, _, err = mcpUploadBody(tool.UploadField, badJSON)
		require.Error(t, err)
	}
}

func TestMCPFullOperationsReceiptsNeverPersistEditableSecrets(t *testing.T) {
	mockDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := mockDB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	previousDB, previousRedis := model.DB, common.RedisEnabled
	model.DB, common.RedisEnabled = mockDB, false
	t.Cleanup(func() { model.DB = previousDB; common.RedisEnabled = previousRedis; _ = sqlDB.Close() })
	require.NoError(t, mockDB.AutoMigrate(&model.User{}, &model.MCPKey{}, &model.MCPWriteRequest{}, &model.MCPCallLog{}))
	user := model.User{Username: "operation-root", Role: common.RoleRootUser, Status: common.UserStatusEnabled}
	require.NoError(t, mockDB.Create(&user).Error)
	key := model.MCPKey{UserID: user.Id, KeyHash: "operation-receipt-test-hash", Tools: []string{"newapi_notification_preview", "newapi_plugin_upload", "newapi_extensions_refresh", "newapi_deployment_logs", "newapi_plugin_source_get", "newapi_extension_notification_publish"}, ExpiresAt: time.Now().Unix() + 3600}
	require.NoError(t, mockDB.Create(&key).Error)
	identity := &MCPIdentity{User: user.ToBaseUser(), Key: &key}
	secret := "api_key=private-operation-fixture Bearer private-operation-token"
	for _, tc := range []struct {
		name string
		args map[string]any
		data any
		read bool
	}{
		{"newapi_notification_preview", map[string]any{"body": map[string]any{"event_type": "channel_disabled", "template": secret}}, map[string]any{"html": secret, "text_length": len(secret)}, true},
		{"newapi_deployment_logs", map[string]any{"path": map[string]string{"id": "cluster-1"}, "query": map[string]string{"container_id": "worker-2"}}, "正常启动\n" + secret, true},
		{"newapi_plugin_source_get", map[string]any{"path": map[string]string{"key": "fixture-plugin"}}, map[string]any{"key": "fixture-plugin", "version": "1.0.0", "source": secret}, true},
		{"newapi_extension_notification_publish", map[string]any{"request_id": "publish-event", "path": map[string]string{"id": "orders"}, "body": map[string]any{"event_type": "created", "event_key": "order:1", "payload": map[string]any{"content": secret}}}, map[string]any{"status": "queued", "delivery_count": 1, "event_type": "extension.orders.created", "event_key": secret, "payload": secret}, false},
		{"newapi_plugin_upload", map[string]any{"request_id": "upload-plugin", "body": map[string]any{"source": "exports.meta={}", "remark": secret}}, map[string]any{"key": "fixture-plugin", "version": "1.0.0", "enabled": false, "remark": secret, "meta": map[string]any{"description": map[string]string{"zh": secret}}}, false},
		{"newapi_extensions_refresh", map[string]any{"request_id": "refresh-extensions", "body": map[string]any{}}, map[string]any{"modules": []any{map[string]any{"id": "fixture-extension", "version": "1.0.0", "description": secret, "notifications": map[string]any{"events": []any{map[string]any{"default_template": secret}}}}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := common.Marshal(map[string]any{"success": true, "data": tc.data})
			require.NoError(t, err)
			calls := 0
			dispatch := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				_, writeErr := w.Write(payload)
				require.NoError(t, writeErr)
			})
			raw, err := common.Marshal(tc.args)
			require.NoError(t, err)
			result := ExecuteMCPTool(context.Background(), identity, tc.name, raw, dispatch)
			require.False(t, result.IsError, "%+v", result)
			require.Len(t, result.Content, 1)
			if tc.name == "newapi_notification_preview" || tc.name == "newapi_plugin_source_get" {
				assert.Contains(t, result.Content[0].Text, secret)
			} else {
				assert.NotContains(t, result.Content[0].Text, secret, "写投影自身也应剔除正文")
			}
			replay := ExecuteMCPTool(context.Background(), identity, tc.name, raw, dispatch)
			require.False(t, replay.IsError, "%+v", replay)
			if tc.read {
				assert.Equal(t, 2, calls)
			} else {
				assert.Equal(t, 1, calls)
				assert.NotContains(t, replay.Content[0].Text, secret)
			}
		})
	}
	var writes []model.MCPWriteRequest
	require.NoError(t, mockDB.Find(&writes).Error)
	require.Len(t, writes, 3, "只读预览、源码和部署日志不能创建写回执")
	for _, write := range writes {
		assert.NotContains(t, write.Result, secret)
		assert.NotContains(t, write.Result, "private-operation")
		assert.Equal(t, "done", write.State)
	}
	var logs []model.MCPCallLog
	require.NoError(t, mockDB.Find(&logs).Error)
	encoded, err := common.Marshal(logs)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "private-operation")
	preview := fullOperationTestTool(t, "newapi_notification_preview")
	assert.False(t, preview.Write)
	assert.True(t, preview.Annotations["readOnlyHint"])
	assert.NotContains(t, preview.InputSchema["properties"], "request_id")
}

func TestMCPFullOperationsDeploymentLogsSensitiveContract(t *testing.T) {
	tool := fullOperationTestTool(t, "newapi_deployment_logs")
	assert.Equal(t, common.RoleRootUser, tool.MinRole)
	assert.False(t, tool.Write)
	assert.True(t, tool.Annotations["readOnlyHint"])
	assert.NotContains(t, tool.InputSchema["properties"], "request_id")
	for _, tc := range []struct {
		name    string
		role    int
		grant   string
		visible bool
	}{
		{"Root单独授权", common.RoleRootUser, tool.Name, true},
		{"Root仅部署详情授权", common.RoleRootUser, "newapi_deployment_get", false},
		{"Admin即使持有日志grant", common.RoleAdminUser, tool.Name, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := model.User{Role: tc.role, Status: common.UserStatusEnabled}
			visible := MCPVisibleTools(user.ToBaseUser(), &model.MCPKey{Tools: []string{tc.grant}})
			found := false
			for _, item := range visible {
				found = found || item.Name == tool.Name
			}
			assert.Equal(t, tc.visible, found)
		})
	}
	for _, tc := range []struct {
		name, input, expected string
		redacted, truncated   bool
	}{
		{"普通日志保留", "启动正常\n请求完成", "启动正常\n请求完成", false, false},
		{"多行headers整段隐藏", "启动\nheaders:\nX-Custom: hidden-value\nAuthorization: hidden-token\n\n恢复正常", "启动\n[redacted]\n\n恢复正常", true, false},
		{"多行credentials整段隐藏", "credentials: {\n\"custom\": \"hidden-value\"\n}\n\n恢复", "[redacted]\n\n恢复", true, false},
		{"内联Bearer", "Bearer hidden-value", "[redacted]", true, false},
		{"混合大小写键", "Api_Key=hidden-value", "[redacted]", true, false},
		{"URL凭据", "upstream https://user:hidden-value@example.test/", "[redacted]", true, false},
		{"JSON单行凭据", `{"message":"ok","headers":{"custom":"hidden-value"}}`, "[redacted]", true, false},
		{"JSON转义字段", `{"heade\u0072s":{"custom":"hidden-value"}}`, "[redacted]", true, false},
		{"多行JSON转义字段", "{\n\"creden\\u0074ials\": {\n\"custom\": \"hidden-value\"\n}\n}", "[redacted]", true, false},
		{"逐行JSON转义字段", "启动\n{\"api\\u005fkey\":\"hidden-value\"}\n\n健康", "启动\n[redacted]\n\n健康", true, false},
		{"中文截断", strings.Repeat("中", 6000), strings.Repeat("中", 5461), false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := common.Marshal(map[string]any{"success": true, "data": tc.input})
			require.NoError(t, err)
			got, success, err := ProjectMCPResponse(tool, raw, http.StatusOK)
			require.NoError(t, err)
			require.True(t, success)
			var response struct {
				Data struct {
					Text      string `json:"text"`
					Redacted  bool   `json:"redacted"`
					Truncated bool   `json:"truncated"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(got, &response))
			assert.Equal(t, tc.expected, response.Data.Text)
			assert.Equal(t, tc.redacted, response.Data.Redacted)
			assert.Equal(t, tc.truncated, response.Data.Truncated)
			assert.True(t, utf8.ValidString(response.Data.Text))
			assert.LessOrEqual(t, len(response.Data.Text), 16*1024)
			assert.NotContains(t, string(got), "hidden-")
		})
	}
	_, _, err := ProjectMCPResponse(tool, []byte(`{"success":true,"data":{"text":"unknown-structure"}}`), http.StatusOK)
	require.Error(t, err, "宿主日志响应变形必须失败，不能把对象当任意正文放行")
}

func TestMCPFullOperationsExtensionSensitiveGrants(t *testing.T) {
	for _, name := range []string{"newapi_plugin_source_get", "newapi_extension_notification_publish"} {
		tool := fullOperationTestTool(t, name)
		assert.Equal(t, common.RoleRootUser, tool.MinRole)
		for _, tc := range []struct {
			role    int
			grant   string
			allowed bool
		}{
			{common.RoleRootUser, name, true},
			{common.RoleRootUser, "newapi_plugin_get", false},
			{common.RoleRootUser, "newapi_extensions_inventory", false},
			{common.RoleAdminUser, name, false},
		} {
			user := model.User{Role: tc.role}
			found := false
			for _, visible := range MCPVisibleTools(user.ToBaseUser(), &model.MCPKey{Tools: []string{tc.grant}}) {
				found = found || visible.Name == name
			}
			assert.Equal(t, tc.allowed, found, "%s role=%d grant=%s", name, tc.role, tc.grant)
		}
	}
	source := "const fixture = 'Bearer explicit-source-secret';"
	raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{
		"key": "demo", "version": "1.0.0", "source": source, "credentials": "hidden-auth",
	}})
	require.NoError(t, err)
	for _, name := range []string{"newapi_plugin_source_get", "newapi_plugin_get"} {
		projected, success, err := ProjectMCPResponse(fullOperationTestTool(t, name), raw, http.StatusOK)
		require.NoError(t, err)
		require.True(t, success)
		assert.NotContains(t, string(projected), "hidden-auth")
		if name == "newapi_plugin_source_get" {
			assert.Contains(t, string(projected), source)
		} else {
			assert.NotContains(t, string(projected), "explicit-source-secret")
		}
	}
}

func TestMCPFullOperationsWriteIdentityProjectionExamples(t *testing.T) {
	for _, tc := range []struct{ name, raw, expected string }{
		{"newapi_extension_upload", `{"success":true,"data":{"root":"hidden","module":{"id":"demo","version":"1.0.0","enabled":false,"asset_revision":"rev1","description":"hidden"}}}`, `{"success":true,"data":{"module":{"id":"demo","version":"1.0.0","enabled":false,"asset_revision":"rev1"}}}`},
		{"newapi_extensions_refresh", `{"success":true,"data":{"modules":[{"id":"demo","version":"1.0.0","enabled":true,"asset_revision":"rev1","description":"hidden"}]}}`, `{"success":true,"data":{"modules":[{"id":"demo","version":"1.0.0","enabled":true,"asset_revision":"rev1"}]}}`},
		{"newapi_extension_uninstall", `{"success":true,"data":{"root":"hidden","modules":[]}}`, `{"success":true,"data":{"modules":[]}}`},
		{"newapi_extension_enabled_set", `{"success":true,"data":{"id":"demo","version":"1.0.0","enabled":true,"asset_revision":"rev1","description":"hidden"}}`, `{"success":true,"data":{"id":"demo","version":"1.0.0","enabled":true,"asset_revision":"rev1"}}`},
		{"newapi_plugin_upload", `{"success":true,"data":{"key":"demo","version":"1.0.0","api_version":1,"source_hash":"hash","enabled":false,"active":false,"source_kind":"custom","channel_count":0,"in_flight_count":0,"source":"hidden","remark":"hidden"}}`, `{"success":true,"data":{"key":"demo","version":"1.0.0","api_version":1,"source_hash":"hash","enabled":false,"active":false,"source_kind":"custom","channel_count":0,"in_flight_count":0}}`},
		{"newapi_plugin_activate", `{"success":true,"data":null}`, `{"success":true,"data":null}`},
		{"newapi_plugin_status_set", `{"success":true,"data":null}`, `{"success":true,"data":null}`},
		{"newapi_extension_notification_publish", `{"success":true,"data":{"status":"queued","delivery_count":1,"event_type":"extension.demo.created","event_key":"hidden"}}`, `{"success":true,"data":{"status":"queued","delivery_count":1,"event_type":"extension.demo.created"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, success, err := ProjectMCPResponse(fullOperationTestTool(t, tc.name), []byte(tc.raw), http.StatusOK)
			require.NoError(t, err)
			require.True(t, success)
			assert.JSONEq(t, tc.expected, string(got), "供核心回执适配的固定 projected 合同")
		})
	}
}
