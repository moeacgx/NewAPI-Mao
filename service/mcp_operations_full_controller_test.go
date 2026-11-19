package service_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/extension"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMCPFullOperationsExtensionPublishAndPluginSourceControllers(t *testing.T) {
	mockDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := mockDB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	previousDB, previousManager := model.DB, extension.DefaultManager
	model.DB = mockDB
	manager := extension.NewManager(t.TempDir())
	extension.DefaultManager = manager
	t.Cleanup(func() { model.DB = previousDB; extension.DefaultManager = previousManager; _ = sqlDB.Close() })
	require.NoError(t, mockDB.AutoMigrate(&model.NotificationBot{}, &model.NotificationTask{}, &model.NotificationTarget{},
		&model.NotificationEvent{}, &model.NotificationEventReceipt{}, &model.NotificationDelivery{},
		&model.TaskPlugin{}, &model.Channel{}, &model.Task{}))
	moduleDir := filepath.Join(manager.RootDir(), "orders")
	require.NoError(t, os.MkdirAll(filepath.Join(moduleDir, "public"), 0o755))
	manifest := extension.Manifest{ID: "orders", Name: "订单测试", Version: "1.0.0",
		Runtime:     extension.Runtime{Type: extension.RuntimeTypeStatic, StaticDir: "public"},
		Permissions: extension.PermissionConfig{Roles: []string{"root"}, Capabilities: []string{extension.CapabilityNotificationEventsPublish}},
		Notifications: extension.NotificationContribution{Events: []extension.NotificationEventContribution{{
			ID: "created", Label: "订单创建", DefaultTemplate: "{{order_id}}",
			Variables: []extension.NotificationVariable{{Name: "order_id", Type: "string", Required: true}, {Name: "amount", Type: "number"}, {Name: "urgent", Type: "boolean"}},
		}}},
	}
	manifestJSON, err := common.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "manifest.json"), manifestJSON, 0o644))
	require.NoError(t, manager.Scan())
	_, err = manager.SetEnabled("orders", true)
	require.NoError(t, err)
	bot := model.NotificationBot{Name: "隔离机器人", Enabled: true}
	require.NoError(t, mockDB.Create(&bot).Error)
	task := model.NotificationTask{Name: "隔离规则", BotId: bot.Id, EventType: "extension.orders.created", Enabled: true, Template: "{{order_id}}"}
	require.NoError(t, mockDB.Create(&task).Error)
	require.NoError(t, mockDB.Create(&model.NotificationTarget{TaskId: task.Id, ChatId: "local-test", Enabled: true}).Error)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("role", common.RoleRootUser); c.Next() })
	engine.POST("/api/extensions/:id/notification-events", controller.PublishExtensionNotificationEvent)
	engine.GET("/api/plugin/task/:key", controller.GetTaskPlugin)
	args := map[string]any{"request_id": "publish", "path": map[string]string{"id": "orders"},
		"body": map[string]any{"event_type": "created", "event_key": "order:1", "payload": map[string]any{"order_id": "1", "amount": 0, "urgent": false}}}
	published := mcpOperationControllerCall(t, engine, "newapi_extension_notification_publish", args)
	assert.Equal(t, map[string]any{"status": "queued", "event_type": "extension.orders.created", "delivery_count": float64(1)}, published["data"])
	duplicate := mcpOperationControllerCall(t, engine, "newapi_extension_notification_publish", args)
	assert.Equal(t, "duplicate", duplicate["data"].(map[string]any)["status"])
	var storedEvent model.NotificationEvent
	require.NoError(t, mockDB.First(&storedEvent).Error)
	var payload map[string]any
	require.NoError(t, common.UnmarshalJsonStr(storedEvent.Payload, &payload))
	assert.Equal(t, float64(0), payload["amount"])
	assert.Equal(t, false, payload["urgent"])
	var deliveries []model.NotificationDelivery
	require.NoError(t, mockDB.Find(&deliveries).Error)
	require.Len(t, deliveries, 1)
	assert.Equal(t, model.NotificationDeliveryPending, deliveries[0].Status, "仅入队，不启动调度或发送消息")
	// 真实插件控制器编译历史版本并校验 hash，普通详情投影仍不得读到源码。
	source := `export const meta={apiVersion:1,key:"mcp-source-fixture",name:"源码测试",version:"1.0.0",author:{name:"Test"},models:["fixture"],fetchMode:"per_task"};
export function buildSubmitRequest(){return {};}
export function parseSubmitResponse(){return {};}
export function buildQueryRequest(){return {};}
export function parseTaskResult(){return {};}
// Bearer source-fixture-sensitive`
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(source)))
	require.NoError(t, mockDB.Create(&model.TaskPlugin{Key: "mcp-source-fixture", Version: "1.0.0", APIVersion: 1, Source: source, SourceHash: hash, SourceKind: "custom"}).Error)
	for _, name := range []string{"newapi_plugin_source_get", "newapi_plugin_get"} {
		got := mcpOperationControllerCall(t, engine, name, map[string]any{"path": map[string]string{"key": "mcp-source-fixture"}, "query": map[string]string{"version": "1.0.0"}})
		data := got["data"].(map[string]any)
		assert.Equal(t, hash, data["source_hash"])
		if name == "newapi_plugin_source_get" {
			assert.Equal(t, source, data["source"])
		} else {
			assert.NotContains(t, data, "source")
		}
	}
}

type mcpOperationProviderTransport func(*http.Request) (*http.Response, error)

func (transport mcpOperationProviderTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestMCPFullOperationsDeploymentControllerContract(t *testing.T) {
	// 真实控制器与 io.net 客户端走内存 transport；不监听端口，不访问真实供应商。
	originalTransport := http.DefaultTransport
	common.OptionMapRWMutex.Lock()
	originalOptions := common.OptionMap
	common.OptionMap = map[string]string{"model_deployment.ionet.enabled": "true", "model_deployment.ionet.api_key": "fixture-provider-key"}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptions
		common.OptionMapRWMutex.Unlock()
	})
	calls := 0
	http.DefaultTransport = mcpOperationProviderTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		assert.Equal(t, "api.io.solutions", req.URL.Host)
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "fixture-provider-key", req.Header.Get("X-API-KEY"))
		var payload string
		switch req.URL.Path {
		case "/v1/io-cloud/caas/deployment/cluster-1/log/worker-2":
			assert.Equal(t, "25", req.URL.Query().Get("limit"))
			assert.Equal(t, "stderr", req.URL.Query().Get("stream"))
			assert.Equal(t, "error", req.URL.Query().Get("level"))
			assert.Equal(t, "cursor-1", req.URL.Query().Get("cursor"))
			assert.Equal(t, "2026-11-15T01:00:00Z", req.URL.Query().Get("start_time"))
			assert.Equal(t, "2026-11-15T02:00:00Z", req.URL.Query().Get("end_time"))
			assert.Empty(t, req.URL.Query().Get("follow"))
			payload = "启动完成\nheaders:\nX-Custom: private-provider-value\n\n部署健康"
		case "/enterprise/v1/io-cloud/caas/hardware/max-gpus-per-container":
			payload = `{"data":{"hardware":[{"hardware_id":1,"available":3}],"total":3}}`
		default:
			return nil, fmt.Errorf("拒绝未约定的供应商请求路径：%s", req.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload)), Request: req}, nil
	})
	engine := gin.New()
	engine.GET("/api/deployments/:id/logs", controller.GetDeploymentLogs)
	engine.POST("/api/deployments/settings/test-connection", controller.TestIoNetConnection)
	engine.POST("/api/deployments/test-connection", controller.TestIoNetConnection)
	logs := mcpOperationControllerCall(t, engine, "newapi_deployment_logs", map[string]any{
		"path":  map[string]string{"id": "cluster-1"},
		"query": map[string]string{"container_id": "worker-2", "limit": "25", "stream": "stderr", "level": "error", "cursor": "cursor-1", "follow": "false", "start_time": "2026-11-15T01:00:00Z", "end_time": "2026-11-15T02:00:00Z"},
	})
	assert.Equal(t, map[string]any{"text": "启动完成\n[redacted]\n\n部署健康", "redacted": true, "truncated": false}, logs["data"])
	for _, name := range []string{"newapi_deployment_settings_test_connection", "newapi_deployment_test_connection"} {
		got := mcpOperationControllerCall(t, engine, name, map[string]any{"request_id": name, "body": map[string]any{}})
		assert.Equal(t, map[string]any{"hardware_count": float64(1), "total_available": float64(3)}, got["data"])
	}
	assert.Equal(t, 3, calls)
}

func mcpOperationControllerCall(t *testing.T, engine *gin.Engine, name string, args map[string]any) map[string]any {
	t.Helper()
	var selected service.MCPTool
	for _, tool := range service.MCPTools() {
		if tool.Name == name {
			selected = tool
			break
		}
	}
	require.NotEmpty(t, selected.Name)
	raw, err := common.Marshal(args)
	require.NoError(t, err)
	_, path, body, err := service.PrepareMCPRequest(selected, raw)
	require.NoError(t, err)
	req := httptest.NewRequest(selected.Method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	projected, success, err := service.ProjectMCPResponse(selected, recorder.Body.Bytes(), recorder.Code)
	require.NoError(t, err)
	require.True(t, success, "真实控制器返回：%s", recorder.Body.String())
	var response map[string]any
	require.NoError(t, common.Unmarshal(projected, &response))
	return response
}

func TestMCPFullOperationsNotificationControllerRoundTrip(t *testing.T) {
	// 独立内存数据库替代全局 DB，不读取 .env 或生产配置，不启动通知调度。
	mockDB, err := gorm.Open(sqlite.Open("file:mcp-full-operation-notifications?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := mockDB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	originalDB, originalType, originalSecret := model.DB, common.MainDatabaseType(), common.CryptoSecret
	model.DB = mockDB
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.CryptoSecret = "mcp-operation-isolated-test-secret"
	t.Cleanup(func() {
		model.DB = originalDB
		common.SetMainDatabaseType(originalType)
		common.CryptoSecret = originalSecret
		_ = sqlDB.Close()
	})
	require.NoError(t, mockDB.AutoMigrate(&model.NotificationBot{}, &model.NotificationTask{}, &model.NotificationTarget{}, &model.NotificationEvent{}, &model.NotificationEventReceipt{}, &model.NotificationDelivery{}))
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("role", common.RoleRootUser); c.Next() })
	engine.POST("/api/notification/bots", controller.CreateNotificationBot)
	engine.GET("/api/notification/bots", controller.ListNotificationBots)
	engine.POST("/api/notification/tasks", controller.CreateNotificationTask)
	engine.GET("/api/notification/tasks", controller.ListNotificationTasks)
	engine.PUT("/api/notification/tasks/:id", controller.UpdateNotificationTask)
	engine.DELETE("/api/notification/tasks/:id", controller.DisableNotificationTask)
	engine.POST("/api/notification/preview", controller.PreviewNotificationTemplate)
	botResponse := mcpOperationControllerCall(t, engine, "newapi_notification_bot_create", map[string]any{"request_id": "bot-create", "body": map[string]any{"name": "检测机器人", "token": "write-only-test-token", "enabled": false}})
	bot := botResponse["data"].(map[string]any)
	assert.True(t, bot["token_configured"].(bool))
	assert.NotContains(t, bot, "token")
	var storedBot model.NotificationBot
	require.NoError(t, mockDB.First(&storedBot).Error)
	assert.NotEqual(t, "write-only-test-token", storedBot.Token)
	taskBody := map[string]any{"name": "故障提醒", "bot_id": storedBot.Id, "event_type": "channel_disabled", "template": "<b>{{channel_name}}</b>", "enabled": false, "targets": []any{map[string]any{"chat_id": "-1001", "mention_name": "值班人员", "enabled": true}}, "filter_config": map[string]any{"status_codes": "429", "error_keywords": []string{"余额不足"}, "prefix_dedup_seconds": 60}}
	created := mcpOperationControllerCall(t, engine, "newapi_notification_task_create", map[string]any{"request_id": "task-create", "body": taskBody})
	id := fmt.Sprintf("%.0f", created["data"].(float64))
	listed := mcpOperationControllerCall(t, engine, "newapi_notification_tasks_edit_list", map[string]any{})
	item := listed["data"].([]any)[0].(map[string]any)
	assert.Equal(t, taskBody["template"], item["template"])
	assert.Equal(t, "-1001", item["targets"].([]any)[0].(map[string]any)["chat_id"])
	assert.Equal(t, "429", item["filter_config"].(map[string]any)["status_codes"])
	taskBody["template"] = "提醒：{{channel_name}}"
	mcpOperationControllerCall(t, engine, "newapi_notification_task_update", map[string]any{"request_id": "task-update", "path": map[string]string{"id": id}, "body": taskBody})
	var storedTask model.NotificationTask
	require.NoError(t, mockDB.First(&storedTask).Error)
	assert.Equal(t, taskBody["template"], storedTask.Template)
	preview := mcpOperationControllerCall(t, engine, "newapi_notification_preview", map[string]any{"body": map[string]any{"event_type": "channel_disabled", "template": taskBody["template"]}})
	assert.Contains(t, preview["data"].(map[string]any)["html"], "提醒：")
	mcpOperationControllerCall(t, engine, "newapi_notification_task_disable", map[string]any{"request_id": "disable", "path": map[string]string{"id": id}})
	var count int64
	require.NoError(t, mockDB.Model(&model.NotificationTask{}).Count(&count).Error)
	assert.EqualValues(t, 1, count, "DELETE 是停用，不是物理删除")
	require.NoError(t, mockDB.Model(&model.NotificationDelivery{}).Count(&count).Error)
	assert.Zero(t, count, "配置和预览测试不能投递真实通知")
}
