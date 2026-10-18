package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const nativeTaskPluginTestSource = `
export const meta = {
 apiVersion: 1, key: "native-test", name: "Native", version: "1.0.0",
 author: {name: "Test"}, models: ["jev-1.13.0"], fetchMode: "per_task",
 routes: [{method: "POST", path: "/native-test/v1/systemone", type: "submit", decode: "decode", render: "render", retainResult: false, models: ["jev-1.13.0"]}]
};
export function buildSubmitRequest() { return {}; }
export function parseSubmitResponse() { return {}; }
export function buildQueryRequest() { return {}; }
export function parseTaskResult() { return {}; }
export const native = {
 decode: function(ctx) {
   if (ctx.body.value.state === "throw") throw new Error("secret-provider-token");
   return {kind: "submit", model: ctx.body.value.model, action: "evaluate", requestBody: {state: ctx.body.value.state, model: ctx.body.value.model}};
 },
 render: function(ctx, task) { return task; },
 error: function(ctx, err) { return {error: err}; }
};`

func setupNativeTaskPluginTest(t *testing.T) *model.TaskPlugin {
	t.Helper()
	oldDB, oldRegistry := model.DB, jsplugin.DefaultRegistry
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.TaskPlugin{}, &model.Option{}))
	model.DB = db
	jsplugin.DefaultRegistry = jsplugin.NewRegistry()
	jsplugin.DefaultRegistry.SetEnabled(false)
	t.Cleanup(func() {
		model.DB, jsplugin.DefaultRegistry = oldDB, oldRegistry
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
	})
	_, err = jsplugin.DefaultRegistry.RegisterFactory(nativeTaskPluginTestSource, jsplugin.Options{})
	require.NoError(t, err)
	hash := sha256.Sum256([]byte(nativeTaskPluginTestSource))
	row := &model.TaskPlugin{Key: "native-test", Version: "1.0.0", APIVersion: 1, Source: nativeTaskPluginTestSource, SourceHash: hex.EncodeToString(hash[:]), SourceKind: "custom"}
	require.NoError(t, db.Create(row).Error)
	return row
}

func TestNativeTaskPluginRoutesRequireDatabaseActivation(t *testing.T) {
	row := setupNativeTaskPluginTest(t)
	require.NoError(t, service.RefreshTaskPluginRoutes())
	assert.Empty(t, jsplugin.DefaultRegistry.Generation().Routes())
	require.NoError(t, model.DB.Create(&model.Option{Key: setting.TaskPluginEnabledKey, Value: "true"}).Error)
	require.NoError(t, service.RefreshTaskPluginRoutes())
	assert.Empty(t, jsplugin.DefaultRegistry.Generation().Routes(), "工厂版本不能绕过数据库激活")
	require.NoError(t, model.ActivateTaskPlugin(row.Key, row.Version))
	require.NoError(t, service.RefreshTaskPluginRoutes())
	binding, ok := jsplugin.DefaultRegistry.Generation().LookupDeclaredRoute(http.MethodPost, "/native-test/v1/systemone")
	require.True(t, ok)
	require.NotNil(t, binding.Route.RetainResult)
	assert.False(t, *binding.Route.RetainResult)
	boundSetting, wrongSetting := `{"task_plugin_key":"native-test"}`, `{"task_plugin_key":"different"}`
	bound := &model.Channel{Type: constant.ChannelTypeTaskPlugin, Setting: &boundSetting}
	wrong := &model.Channel{Type: constant.ChannelTypeTaskPlugin, Setting: &wrongSetting}
	assert.True(t, model.TaskPluginChannelMatchesPath(bound, binding.Route.Path))
	assert.False(t, model.TaskPluginChannelMatchesPath(wrong, binding.Route.Path))
	assert.False(t, model.TaskPluginChannelMatchesPath(&model.Channel{Type: constant.ChannelTypeOpenAI}, binding.Route.Path))
	assert.False(t, model.TaskPluginChannelMatchesPath(bound, "/v1/chat/completions"))
	require.NoError(t, model.SetTaskPluginEnabled(row.Key, false))
	require.NoError(t, service.RefreshTaskPluginRoutes())
	assert.Empty(t, jsplugin.DefaultRegistry.Generation().Routes())
}

func TestNativeTaskPluginDecodeAndStalePin(t *testing.T) {
	row := setupNativeTaskPluginTest(t)
	require.NoError(t, model.DB.Create(&model.Option{Key: setting.TaskPluginEnabledKey, Value: "true"}).Error)
	require.NoError(t, model.ActivateTaskPlugin(row.Key, row.Version))
	require.NoError(t, service.RefreshTaskPluginRoutes())
	generation := jsplugin.DefaultRegistry.Generation()
	binding, ok := generation.LookupDeclaredRoute(http.MethodPost, "/native-test/v1/systemone")
	require.True(t, ok)
	router := gin.New()
	router.POST(binding.Route.Path, func(c *gin.Context) {
		c.Set(jsplugin.ContextKeyPinnedRoute, jsplugin.PinnedRoute{Generation: generation, Plugin: binding.Plugin, Route: binding.Route})
		c.Next()
	}, PrepareTaskPluginRoute(), func(c *gin.Context) {
		assert.Equal(t, "jev-1.13.0", c.GetString("resolved_task_model"))
		assert.Equal(t, "evaluate", c.GetString("task_action"))
		assert.Equal(t, binding.Plugin, c.MustGet("official_task_plugin"))
		pin := c.MustGet("task_plugin_snapshot").(*model.TaskPluginSnapshot)
		assert.Equal(t, row.SourceHash, pin.SourceHash)
		c.JSON(http.StatusOK, c.MustGet("task_request"))
	})
	for _, test := range []struct {
		name, body, contentType string
		status                  int
	}{
		{"decoded", `{"model":"jev-1.13.0","state":"ok","ignored":true}`, "application/json", http.StatusOK},
		{"wrong model", `{"model":"other","state":"ok"}`, "application/json", http.StatusBadRequest},
		{"hook error", `{"model":"jev-1.13.0","state":"throw"}`, "application/json", http.StatusBadRequest},
		{"invalid body", `[]`, "application/json", http.StatusBadRequest},
		{"unsupported content", `{}`, "text/plain", http.StatusUnsupportedMediaType},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, binding.Route.Path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			assert.Equal(t, test.status, response.Code, response.Body.String())
			assert.NotContains(t, response.Body.String(), "secret-provider-token")
			assert.NotContains(t, response.Body.String(), "ignored")
		})
	}
	// 路由已经固定后禁用插件，仍必须在请求进入插件前拒绝。
	require.NoError(t, model.SetTaskPluginEnabled(row.Key, false))
	request := httptest.NewRequest(http.MethodPost, binding.Route.Path, strings.NewReader(`{"model":"jev-1.13.0","state":"ok"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assert.Equal(t, http.StatusServiceUnavailable, response.Code)
}

func TestNativeTaskPluginErrorRendererDoesNotReceiveProviderDetails(t *testing.T) {
	setupNativeTaskPluginTest(t)
	loaded, err := jsplugin.CompilePlugin(nativeTaskPluginTestSource, jsplugin.Options{})
	require.NoError(t, err)
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodPost, "/native-test/v1/systemone", nil)
	c.Set(jsplugin.ContextKeyPinnedRoute, jsplugin.PinnedRoute{Plugin: loaded})
	c.Set(jsplugin.ContextKeyRouteRequest, jsplugin.RouteRequestContext{})
	require.True(t, RespondTaskPluginError(c, &dto.TaskError{StatusCode: http.StatusBadRequest, Message: "secret-provider-token"}))
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.NotContains(t, response.Body.String(), "secret-provider-token")
	assert.Contains(t, response.Body.String(), "invalid_request")
}

func TestNativeTaskPluginUpstreamsManifestContract(t *testing.T) {
	for _, test := range []struct {
		name, metadata string
		valid, newAPI  bool
	}{
		{"omitted implies vendor", "", true, false},
		{"vendor only", `upstreams:["vendor"],`, true, false},
		{"vendor and new api", `upstreams:["vendor","new_api"],`, true, true},
		{"new api implies vendor", `upstreams:["new_api"],`, true, true},
		{"unknown kind", `upstreams:["gateway"],`, false, false},
		{"duplicate", `upstreams:["new_api","new_api"],`, false, false},
		{"null", `upstreams:null,`, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(nativeTaskPluginTestSource, "apiVersion: 1,", "apiVersion: 1,"+test.metadata, 1)
			plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
			if !test.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, plugin.Meta.SupportsUpstream(jsplugin.UpstreamKindVendor))
			assert.Equal(t, test.newAPI, plugin.Meta.SupportsUpstream(jsplugin.UpstreamKindNewAPI))
		})
	}
}
