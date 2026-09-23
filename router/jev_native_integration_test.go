package router

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 官方插件测试夹具逐字保留 QuantumNous/new-api-plugins 的 Apache-2.0 源码。
// 来源固定为 b42cc99a6bd1998d0cc1581270bd46798ce00ad6；不修改插件以迁就宿主。
func TestJevNativeIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, i18n.Init())
	db := setupFeatureRouterAuthTest(t)
	model.InitDBColumns()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.TaskPlugin{}, &model.Channel{}, &model.Token{}, &model.Log{}, &model.Group{}, &model.GroupAlias{}, &model.ChannelGroupBinding{}, &model.Ability{}, &model.UserSubscription{}, &model.SubscriptionPreConsumeRecord{}, &model.PromptAuditConfig{}, &model.RequestArchiveConfig{}, &model.PromptAuditQueueState{}, &model.RequestArchiveQueueState{}))
	oldRegistry, oldLogDB := jsplugin.DefaultRegistry, model.LOG_DB
	oldCache, oldBatch, oldLog := common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled
	oldQuotaPerUnit := common.QuotaPerUnit
	jsplugin.DefaultRegistry = jsplugin.NewRegistry()
	model.LOG_DB = db
	common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled = false, false, false
	common.QuotaPerUnit = 500000
	t.Cleanup(func() {
		jsplugin.DefaultRegistry = oldRegistry
		model.LOG_DB = oldLogDB
		common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled = oldCache, oldBatch, oldLog
		common.QuotaPerUnit = oldQuotaPerUnit
	})
	savedBilling := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		if strings.HasPrefix(key, "billing_setting.") {
			savedBilling[key] = value
		}
		return nil
	}))
	t.Cleanup(func() { require.NoError(t, config.GlobalConfig.LoadFromDB(savedBilling)) })
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"jev-1.13.0":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"jev-1.13.0":"u(\"input_tokens\") * 0.042 / 1000000"}`,
	}))
	source, err := os.ReadFile("testdata/typesafe-1.0.0.js")
	require.NoError(t, err)
	loaded, err := jsplugin.CompilePlugin(string(source), jsplugin.Options{})
	require.NoError(t, err)
	plugin := model.TaskPlugin{Key: loaded.Meta.Key, Version: loaded.Meta.Version, APIVersion: loaded.Meta.APIVersion, Source: string(source), SourceHash: fmt.Sprintf("%x", sha256.Sum256(source)), SourceKind: "custom", Active: true, Enabled: true}
	require.NoError(t, db.Create(&plugin).Error)
	require.NoError(t, db.Create(&model.Option{Key: "TaskPluginEnabled", Value: "true"}).Error)
	group := model.Group{Code: "default", Name: "Default", Status: model.GroupStatusActive, UserSelectable: true, Ratio: 1}
	require.NoError(t, db.Create(&group).Error)
	user := model.User{Id: 93201, Username: "jev-native-integration", Quota: 1000000, Status: common.UserStatusEnabled, Group: group.Code, Setting: `{"billing_preference":"wallet_only"}`}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{Id: 93201, UserId: user.Id, Key: "jevnativeintegrationtoken", Status: common.TokenStatusEnabled, ExpiredTime: -1, RemainQuota: 1000000}
	require.NoError(t, db.Create(&token).Error)
	var calls atomic.Int32
	var upstreamStatus atomic.Int32
	upstreamStatus.Store(http.StatusOK)
	const answer = `{"model":"jev-1.13.0","answers":{"urgent":{"type":"noul","noul":0},"department":{"type":"choice","choice":"billing","confidence":1,"probabilities":{"billing":1,"technical":0}},"severity":{"type":"score","score":0.25,"confidence":0.5,"legend":{"0":{"label":"低"},"1":{"label":"高"}},"probabilities":{"0":0.75,"1":0.25}}},"usage":{"input_tokens":1000,"output_tokens":73}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/systemone", r.URL.Path)
		assert.Equal(t, "Bearer jev-provider-secret", r.Header.Get("Authorization"))
		var body map[string]any
		if !assert.NoError(t, common.DecodeJson(r.Body, &body)) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		assert.Equal(t, "jev-1.13.0", body["model"])
		assert.Contains(t, body, "state")
		assert.Nil(t, body["state"])
		assert.Len(t, body["questions"], 3)
		assert.NotContains(t, body, "stream")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(upstreamStatus.Load()))
		if upstreamStatus.Load() != http.StatusOK {
			_, _ = io.WriteString(w, `{"detail":"PRIVATE_PROVIDER_ERROR"}`)
			return
		}
		_, _ = io.WriteString(w, answer)
	}))
	t.Cleanup(server.Close)
	service.InitHttpClient()
	// 其他 key 和原生渠道故意拥有更高优先级，成功请求必须仍选中 typesafe。
	for _, ch := range []model.Channel{
		{Id: 93201, Type: constant.ChannelTypeTaskPlugin, Name: "jev-target", Priority: common.GetPointer(int64(10)), Key: "jev-provider-secret", BaseURL: &server.URL, Models: "jev-1.13.0", Group: group.Code, Status: common.ChannelStatusEnabled, Setting: common.GetPointer(`{"task_plugin_key":"typesafe"}`)},
		{Id: 93202, Type: constant.ChannelTypeTaskPlugin, Name: "jev-wrong-key", Priority: common.GetPointer(int64(200)), Key: "wrong-key", BaseURL: &server.URL, Models: "jev-1.13.0", Group: group.Code, Status: common.ChannelStatusEnabled, Setting: common.GetPointer(`{"task_plugin_key":"sora"}`)},
		{Id: 93203, Type: constant.ChannelTypeAtlasCloud, Name: "jev-native", Priority: common.GetPointer(int64(300)), Key: "wrong-native-key", BaseURL: &server.URL, Models: "jev-1.13.0", Group: group.Code, Status: common.ChannelStatusEnabled},
	} {
		require.NoError(t, db.Create(&ch).Error)
		require.NoError(t, db.Create(&model.ChannelGroupBinding{ChannelId: ch.Id, GroupId: group.Id}).Error)
		require.NoError(t, db.Create(&model.Ability{ChannelId: ch.Id, Group: group.Code, GroupId: group.Id, Model: "jev-1.13.0", Enabled: true, Priority: ch.Priority}).Error)
	}
	engine := gin.New()
	engine.GET("/v1/task/plugins/:plugin_key/:task_id", middleware.TokenAuth(), controller.GetOfficialPluginTask)
	engine.GET("/v1/task/plugins/:plugin_key/:task_id/artifacts", middleware.TokenAuth(), controller.GetOfficialPluginArtifacts)
	engine.NoRoute(SetPluginRouter(engine), func(c *gin.Context) { c.Status(http.StatusNotFound) })
	require.NoError(t, service.RefreshTaskPluginRoutes())
	const requestBody = `{"model":"jev-1.13.0","state":null,"stream":false,"questions":{"urgent":{"type":"noul"},"department":{"type":"choice","instructions":null,"criteria":{"billing":null,"technical":"故障"}},"severity":{"type":"score","instructions":"严重程度","criteria":[{"label":"低"},{"label":"高"}]}}}`
	send := func(method, path, auth string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		if auth != "" {
			req.Header.Set("Authorization", "Bearer "+auth)
		}
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		return rec
	}
	t.Run("真实令牌认证阻止匿名调用", func(t *testing.T) {
		rec := send(http.MethodPost, "/typesafe/v1/systemone", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
		assert.EqualValues(t, 0, calls.Load())
	})
	var successTask model.Task
	t.Run("三题型同步返回并按实际输入用量结算", func(t *testing.T) {
		rec := send(http.MethodPost, "/typesafe/v1/systemone", token.Key)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.JSONEq(t, answer, rec.Body.String())
		assert.EqualValues(t, 1, calls.Load())
		require.NoError(t, db.First(&successTask).Error)
		assert.Equal(t, 93201, successTask.ChannelId)
		assert.EqualValues(t, model.TaskStatusSuccess, successTask.Status)
		assert.Equal(t, 21, successTask.Quota)
		var actualUser model.User
		var actualToken model.Token
		require.NoError(t, db.First(&actualUser, user.Id).Error)
		require.NoError(t, db.First(&actualToken, token.Id).Error)
		assert.EqualValues(t, 1000000-21, actualUser.Quota)
		assert.EqualValues(t, 1000000-21, actualToken.RemainQuota)
		assert.EqualValues(t, 21, actualToken.UsedQuota)
	})
	t.Run("不保留结果但保留账务及版本归因", func(t *testing.T) {
		require.NotEmpty(t, successTask.TaskID)
		assert.True(t, successTask.PrivateData.ResultDiscarded)
		assert.Empty(t, successTask.PrivateData.PluginData)
		assert.Empty(t, successTask.PrivateData.PluginState)
		assert.Empty(t, successTask.PrivateData.Key)
		assert.Nil(t, successTask.PrivateData.PluginImmediate)
		require.NotNil(t, successTask.PrivateData.Execution)
		require.NotNil(t, successTask.PrivateData.Execution.TaskPlugin)
		assert.Equal(t, plugin.SourceHash, successTask.PrivateData.Execution.TaskPlugin.SourceHash)
		assert.NotContains(t, string(successTask.Data), "answers")
		for _, suffix := range []string{"", "/artifacts"} {
			rec := send(http.MethodGet, "/v1/task/plugins/typesafe/"+successTask.TaskID+suffix, token.Key)
			assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
			assert.NotContains(t, rec.Body.String(), "answers")
		}
	})
	t.Run("上游错误退回预扣且不泄露错误正文", func(t *testing.T) {
		upstreamStatus.Store(http.StatusBadGateway)
		rec := send(http.MethodPost, "/typesafe/v1/systemone", token.Key)
		assert.Equal(t, http.StatusBadGateway, rec.Code, rec.Body.String())
		assert.NotContains(t, rec.Body.String(), "PRIVATE_PROVIDER_ERROR")
		assert.EqualValues(t, 2, calls.Load())
		var actualUser model.User
		var actualToken model.Token
		// 退款由既有资金会话异步提交，等待账务结果而非假定 HTTP 返回即完成。
		require.Eventually(t, func() bool {
			return db.First(&actualUser, user.Id).Error == nil &&
				db.First(&actualToken, token.Id).Error == nil &&
				actualUser.Quota == 1000000-21 && actualToken.RemainQuota == 1000000-21 && actualToken.UsedQuota == 21
		}, 2*time.Second, 10*time.Millisecond)
		assert.EqualValues(t, 1000000-21, actualUser.Quota)
		assert.EqualValues(t, 1000000-21, actualToken.RemainQuota)
		assert.EqualValues(t, 21, actualToken.UsedQuota)
		var count int64
		require.NoError(t, db.Model(&model.Task{}).Count(&count).Error)
		assert.EqualValues(t, 1, count)
	})
	t.Run("任务落库后结算失败保留预留资金", func(t *testing.T) {
		upstreamStatus.Store(http.StatusOK)
		var persisted atomic.Bool
		var settlementUpdates atomic.Int32
		const createdCallback = "test:jev_task_persisted"
		const failureCallback = "test:jev_settlement_failure"
		require.NoError(t, db.Callback().Create().After("gorm:create").Register(createdCallback, func(tx *gorm.DB) {
			if tx.Statement.Table == "tasks" && tx.Error == nil {
				persisted.Store(true)
			}
		}))
		defer func() { require.NoError(t, db.Callback().Create().Remove(createdCallback)) }()
		require.NoError(t, db.Callback().Update().Before("gorm:update").Register(failureCallback, func(tx *gorm.DB) {
			if !persisted.Load() || tx.Statement.Table != "users" {
				return
			}
			changes, ok := tx.Statement.Dest.(map[string]any)
			if !ok {
				return
			}
			if _, changesQuota := changes["quota"]; changesQuota && settlementUpdates.Add(1) == 1 {
				// 只破坏落库之后的第一次结算；若错误地安排全额退款，第二次更新会成功。
				tx.AddError(errors.New("PRIVATE_SETTLEMENT_FAILURE"))
			}
		}))
		defer func() { require.NoError(t, db.Callback().Update().Remove(failureCallback)) }()
		rec := send(http.MethodPost, "/typesafe/v1/systemone", token.Key)
		require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
		assert.NotContains(t, rec.Body.String(), "PRIVATE_SETTLEMENT_FAILURE")
		assert.True(t, persisted.Load())
		// 等待既有异步工作退出，防止错误退款尚未执行而把测试误判为通过。
		require.Eventually(t, func() bool { return gopool.WorkerCount() == 0 }, 2*time.Second, 10*time.Millisecond)
		assert.EqualValues(t, 1, settlementUpdates.Load())
		assert.EqualValues(t, 3, calls.Load())
		var persistedTask model.Task
		require.NoError(t, db.Order("id DESC").First(&persistedTask).Error)
		assert.NotEqual(t, successTask.TaskID, persistedTask.TaskID)
		assert.EqualValues(t, model.TaskStatusSuccess, persistedTask.Status)
		assert.True(t, persistedTask.PrivateData.ResultDiscarded)
		assert.Equal(t, 21, persistedTask.Quota)
		var actualUser model.User
		var actualToken model.Token
		require.NoError(t, db.First(&actualUser, user.Id).Error)
		require.NoError(t, db.First(&actualToken, token.Id).Error)
		assert.EqualValues(t, 1000000-21-1376, actualUser.Quota)
		assert.EqualValues(t, 1000000-21-1376, actualToken.RemainQuota)
		assert.EqualValues(t, 21+1376, actualToken.UsedQuota)
	})
	t.Run("禁用插件与总开关撤销原生入口", func(t *testing.T) {
		require.NoError(t, model.SetTaskPluginEnabled(plugin.Key, false))
		require.NoError(t, service.RefreshTaskPluginRoutes())
		rec := send(http.MethodPost, "/typesafe/v1/systemone", token.Key)
		assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.NoError(t, model.SetTaskPluginEnabled(plugin.Key, true))
		require.NoError(t, db.Model(&model.Option{}).Where("key = ?", "TaskPluginEnabled").Update("value", "false").Error)
		require.NoError(t, service.RefreshTaskPluginRoutes())
		rec = send(http.MethodPost, "/typesafe/v1/systemone", token.Key)
		assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		assert.EqualValues(t, 3, calls.Load())
	})
}
