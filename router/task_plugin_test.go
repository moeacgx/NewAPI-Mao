package router

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskPluginManagementContractAndPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFeatureRouterAuthTest(t)
	model.InitDBColumns()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	oldLogDB := model.LOG_DB
	model.LOG_DB = db
	t.Cleanup(func() {
		require.Eventually(t, func() bool { var count int64; return db.Model(&model.Log{}).Count(&count).Error == nil && count >= 4 }, 2*time.Second, 10*time.Millisecond)
		model.LOG_DB = oldLogDB
	})
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.TaskPlugin{}, &model.Channel{}, &model.Log{}))
	require.NoError(t, service.InitTaskPlugins())
	_, admin := issueFeatureRouterSession(t, common.RoleAdminUser, "plugin-admin")
	_, root := issueFeatureRouterSession(t, common.RoleRootUser, "plugin-root")
	_, user := issueFeatureRouterSession(t, common.RoleCommonUser, "plugin-user")
	engine := gin.New()
	registerTaskPluginManagement(engine.Group("/api"))
	for _, token := range []string{"", user} {
		recorder := serveFeatureRouterRequest(engine, http.MethodGet, "/api/plugin/task", token)
		assert.NotEqual(t, http.StatusOK, recorder.Code)
	}
	list := serveFeatureRouterRequest(engine, http.MethodGet, "/api/plugin/task", admin)
	require.Equal(t, http.StatusOK, list.Code)
	var result struct {
		Success bool             `json:"success"`
		Data    []map[string]any `json:"data"`
	}
	require.NoError(t, common.Unmarshal(list.Body.Bytes(), &result))
	require.True(t, result.Success)
	require.Len(t, result.Data, 10)
	keys := map[string]bool{}
	for _, item := range result.Data {
		key := item["key"].(string)
		assert.False(t, keys[key])
		keys[key] = true
		assert.NotEmpty(t, item["meta"])
		assert.Equal(t, false, item["active"])
		assert.Equal(t, false, item["enabled"])
		assert.Equal(t, float64(0), item["channel_count"])
		assert.NotContains(t, item, "source")
	}
	for _, token := range []string{admin, root} {
		detail := serveFeatureRouterRequest(engine, http.MethodGet, "/api/plugin/task/sora", token)
		var parsed struct {
			Success bool           `json:"success"`
			Data    map[string]any `json:"data"`
		}
		require.NoError(t, common.Unmarshal(detail.Body.Bytes(), &parsed))
		require.True(t, parsed.Success)
		assert.NotEmpty(t, parsed.Data["meta"])
		if token == admin {
			assert.NotContains(t, parsed.Data, "source")
		} else {
			assert.NotEmpty(t, parsed.Data["source"])
		}
	}
	for _, path := range []string{"/api/plugin/task/runtime/status", "/api/plugin/task/sora/status", "/api/plugin/task/sora/activate"} {
		method := http.MethodPost
		if strings.Contains(path, "runtime") {
			method = http.MethodPut
		}
		recorder := serveFeatureRouterRequest(engine, method, path, admin)
		assert.Equal(t, http.StatusForbidden, recorder.Code, path)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/plugin/task/runtime/status", strings.NewReader(`{"enabled":true}`))
	request.Header.Set("Authorization", "Bearer "+root)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	enabled, err := service.TaskPluginsEnabled()
	require.NoError(t, err)
	assert.True(t, enabled)
	// enabled=true 仍不能绕过首次激活要求。
	require.NoError(t, db.Model(&model.TaskPlugin{}).Where("key = ?", "sora").Update("enabled", true).Error)
	_, _, err = service.ActiveTaskPlugin("sora")
	assert.Error(t, err)
	version, err := model.GetTaskPluginVersion("sora", "")
	require.NoError(t, err)
	require.NoError(t, model.ActivateTaskPlugin("sora", version.Version))
	_, pin, err := service.ActiveTaskPlugin("sora")
	require.NoError(t, err)
	require.NoError(t, model.SetTaskPluginEnabled("sora", false))
	_, _, err = service.ActiveTaskPlugin("sora")
	assert.Error(t, err)
	_, err = service.LoadPinnedTaskPlugin(pin)
	assert.NoError(t, err)
}

func TestTaskPluginMixedChannelRealDistribution(t *testing.T) {
	for _, memory := range []bool{false, true} {
		t.Run(strconv.FormatBool(memory), func(t *testing.T) {
			db := setupFeatureRouterAuthTest(t)
			model.InitDBColumns()
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			require.NoError(t, db.AutoMigrate(&model.TaskPlugin{}, &model.Option{}, &model.Channel{}, &model.Ability{}, &model.Group{}, &model.GroupAlias{}, &model.ChannelGroupBinding{}))
			oldCache := common.MemoryCacheEnabled
			common.MemoryCacheEnabled = memory
			t.Cleanup(func() { common.MemoryCacheEnabled = oldCache })
			group := model.Group{Code: "default", Name: "Default", Ratio: 1, Status: model.GroupStatusActive, UserSelectable: true}
			require.NoError(t, db.Create(&group).Error)
			for _, channel := range []model.Channel{
				{Id: 9101, Type: constant.ChannelTypeAtlasCloud, Name: "native", Priority: common.GetPointer(int64(100)), Key: "test", Models: "sora-2", Group: "default", Status: common.ChannelStatusEnabled},
				{Id: 9102, Type: constant.ChannelTypeTaskPlugin, Name: "target", Priority: common.GetPointer(int64(10)), Key: "test", Models: "sora-2", Group: "default", Status: common.ChannelStatusEnabled, Setting: common.GetPointer(`{"task_plugin_key":"sora"}`)},
				{Id: 9103, Type: constant.ChannelTypeTaskPlugin, Name: "other", Priority: common.GetPointer(int64(200)), Key: "test", Models: "sora-2", Group: "default", Status: common.ChannelStatusEnabled, Setting: common.GetPointer(`{"task_plugin_key":"vidu"}`)},
			} {
				require.NoError(t, db.Create(&channel).Error)
				require.NoError(t, db.Create(&model.ChannelGroupBinding{ChannelId: channel.Id, GroupId: group.Id}).Error)
				require.NoError(t, db.Create(&model.Ability{ChannelId: channel.Id, Group: "default", GroupId: group.Id, Model: "sora-2", Enabled: true, Priority: channel.Priority}).Error)
			}
			if memory {
				model.InitChannelCache()
			}
			require.NoError(t, service.InitTaskPlugins())
			version, err := model.GetTaskPluginVersion("sora", "")
			require.NoError(t, err)
			require.NoError(t, model.ActivateTaskPlugin("sora", version.Version))
			require.NoError(t, db.Create(&model.Option{Key: "TaskPluginEnabled", Value: "true"}).Error)
			engine := gin.New()
			engine.Use(func(c *gin.Context) {
				c.Set("id", 101)
				common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
				common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
				common.SetContextKey(c, constant.ContextKeyTokenGroupMode, model.TokenGroupModeInherit)
				if id := c.GetHeader("X-Test-Channel"); id != "" {
					common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, id)
				}
				c.Next()
			})
			respond := func(c *gin.Context) { c.JSON(200, gin.H{"channel_id": c.GetInt("channel_id")}) }
			engine.POST("/v1/task/plugins/:plugin_key", middleware.TaskPluginRequest(), middleware.Distribute(), respond)
			engine.POST("/v1/videos", middleware.Distribute(), respond)
			engine.POST("/v1/chat/completions", middleware.Distribute(), respond)
			for _, tc := range []struct {
				path, forced string
				id           int
			}{
				{"/v1/task/plugins/sora", "", 9102}, {"/v1/videos", "", 9101}, {"/v1/chat/completions", "", 9101},
				{"/v1/task/plugins/sora", "9101", 0}, {"/v1/task/plugins/sora", "9103", 0}, {"/v1/videos", "9102", 0},
			} {
				req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(`{"model":"sora-2","prompt":"test"}`))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Test-Channel", tc.forced)
				rec := httptest.NewRecorder()
				engine.ServeHTTP(rec, req)
				if tc.id == 0 {
					assert.NotEqual(t, 200, rec.Code, rec.Body.String())
					continue
				}
				require.Equal(t, 200, rec.Code, rec.Body.String())
				var body map[string]int
				require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tc.id, body["channel_id"])
			}
		})
	}
}
