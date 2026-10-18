package router

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/perf_metrics_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试的插件key刻意不含Jev，证明宿主接线不依赖供应商名称。
const taskPerformanceFilterHook = `
export function shouldRecordPerformanceFailure(ctx, failure) {
 if (ctx.model !== "filter-alias" || ctx.upstreamModel !== "token-model" ||
     ctx.method !== "POST" || ctx.requestPath !== "/performance-gate" ||
     Object.keys(ctx).length !== 6 || Object.keys(failure).length !== 3) return true;
 if (failure.stage === "http" && failure.errorCode === "ignore") return false;
 if (failure.stage === "parse" && failure.errorCode === "plugin_submit_response_failed") return false;
 if (failure.stage === "immediate" && failure.errorCode === "123") return false;
 if (failure.stage === "transport") return false;
 if (failure.errorCode === "throw") throw new Error("PRIVATE_HOOK_ERROR");
 if (failure.errorCode === "invalid") return "false";
 return true;
}`

func TestTaskPluginPerformanceFilterPreservesLogsAndBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, i18n.Init())
	db := setupFeatureRouterAuthTest(t)
	model.InitDBColumns()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.TaskPlugin{}, &model.Channel{}, &model.Token{}, &model.Log{}, &model.PerfMetric{}, &model.Group{}, &model.GroupAlias{}, &model.ChannelGroupBinding{}, &model.Ability{}, &model.PromptAuditConfig{}, &model.PromptAuditEndpoint{}, &model.RequestArchiveConfig{}, &model.RequestArchiveTarget{}, &model.PromptAuditQueueState{}, &model.RequestArchiveQueueState{}))
	oldRegistry, oldLogDB := jsplugin.DefaultRegistry, model.LOG_DB
	oldCache, oldBatch, oldLog, oldErrors := common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled, constant.ErrorLogEnabled
	oldPrice := ratio_setting.ModelPrice2JSONString()
	oldGroupRatios := ratio_setting.GroupRatio2JSONString()
	oldPerf := perf_metrics_setting.GetSetting()
	savedBilling := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(k, v string) error {
		if strings.HasPrefix(k, "billing_setting.") {
			savedBilling[k] = v
		}
		return nil
	}))
	jsplugin.DefaultRegistry = jsplugin.NewRegistry()
	model.LOG_DB = db
	common.MemoryCacheEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true
	constant.ErrorLogEnabled = true
	t.Cleanup(func() {
		jsplugin.DefaultRegistry = oldRegistry
		model.LOG_DB = oldLogDB
		common.MemoryCacheEnabled = oldCache
		common.BatchUpdateEnabled = oldBatch
		common.LogConsumeEnabled = oldLog
		constant.ErrorLogEnabled = oldErrors
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(oldPrice))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroupRatios))
		require.NoError(t, config.GlobalConfig.LoadFromDB(savedBilling))
		rules, marshalErr := common.Marshal(oldPerf.FailureFilterRules)
		require.NoError(t, marshalErr)
		require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"perf_metrics_setting.enabled": fmt.Sprint(oldPerf.Enabled), "perf_metrics_setting.failure_filter_rules": string(rules)}))
	})
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"billing_setting.billing_mode": "{}", "perf_metrics_setting.enabled": "true", "perf_metrics_setting.failure_filter_rules": "[]"}))
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"filter-alias":0.001}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"perf-filter":1}`))
	source := strings.ReplaceAll(tokenUsagePluginFixture, "token-usage", "performance-gate")
	source = strings.Replace(source, `models: ["token-model"]`, `models: ["token-model","filter-alias"]`, 1)
	source = strings.Replace(source, "apiVersion: 1,", "apiVersion: 1, requiredCapabilities: [\"task-performance-filter@1\"],", 1)
	source = strings.Replace(source, `immediate:{status:"FAILURE"}`, `immediate:{status:"FAILURE",code:response.body.code}`, 1) + taskPerformanceFilterHook
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.TaskPlugin{Key: plugin.Meta.Key, Version: plugin.Meta.Version, APIVersion: 1, Source: source, SourceHash: fmt.Sprintf("%x", sha256.Sum256([]byte(source))), SourceKind: "custom", Active: true, Enabled: true}).Error)
	require.NoError(t, db.Create(&model.Option{Key: "TaskPluginEnabled", Value: "true"}).Error)
	group := model.Group{Code: "perf-filter", Name: "性能过滤测试", Ratio: 1, Status: model.GroupStatusActive, UserSelectable: true}
	require.NoError(t, db.Create(&group).Error)
	user := model.User{Id: 98731, Username: "performance-filter", Group: group.Code, Quota: 1000000, Status: common.UserStatusEnabled, Setting: `{"billing_preference":"wallet_only"}`}
	require.NoError(t, db.Create(&user).Error)
	token := model.Token{Id: 98731, UserId: user.Id, Key: "performancefiltertest", Status: common.TokenStatusEnabled, ExpiredTime: -1, RemainQuota: 1000000}
	require.NoError(t, db.Create(&token).Error)
	status := http.StatusBadGateway
	body := `{"error":{"code":"ignore","message":"PRIVATE_UPSTREAM_BODY"}}`
	var duringRequest func()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if duringRequest != nil {
			duringRequest()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	service.InitHttpClient()
	channel := model.Channel{Id: 98731, Type: constant.ChannelTypeTaskPlugin, Name: "performance", Key: "PRIVATE_UPSTREAM_KEY", BaseURL: &server.URL, Models: "filter-alias", ModelMapping: common.GetPointer(`{"filter-alias":"token-model"}`), Group: group.Code, Status: common.ChannelStatusEnabled, Setting: common.GetPointer(`{"task_plugin_key":"performance-gate"}`)}
	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, db.Create(&model.ChannelGroupBinding{ChannelId: channel.Id, GroupId: group.Id}).Error)
	require.NoError(t, db.Create(&model.Ability{ChannelId: channel.Id, Group: group.Code, GroupId: group.Id, Model: "filter-alias", Enabled: true}).Error)
	engine := gin.New()
	engine.NoRoute(SetPluginRouter(engine), func(c *gin.Context) { c.Status(404) })
	require.NoError(t, service.RefreshTaskPluginRoutes())
	samples := func() int64 {
		t.Helper()
		summary, queryErr := perfmetrics.QuerySummaryAll(24, []string{group.Code})
		require.NoError(t, queryErr)
		for _, metric := range summary.Models {
			if metric.ModelName == "filter-alias" {
				return metric.RequestCount
			}
		}
		return 0
	}
	count := samples()
	balance := user.Quota
	send := func(name string, wantCode int, delta int64, logDelta int64) {
		t.Helper()
		var oldLogs int64
		require.NoError(t, db.Model(&model.Log{}).Where("type = ?", model.LogTypeError).Count(&oldLogs).Error)
		req := httptest.NewRequest("POST", "/performance-gate?private=query-secret", strings.NewReader(`{"model":"filter-alias"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token.Key)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		assert.Equal(t, wantCode, rec.Code, rec.Body.String())
		assert.NotContains(t, rec.Body.String(), "PRIVATE_")
		if wantCode >= 400 {
			require.Eventually(t, func() bool {
				var u model.User
				var k model.Token
				return db.First(&u, user.Id).Error == nil && db.First(&k, token.Id).Error == nil && u.Quota == balance && int64(k.RemainQuota) == balance
			}, 2*time.Second, 10*time.Millisecond)
		}
		var logs int64
		require.NoError(t, db.Model(&model.Log{}).Where("type = ?", model.LogTypeError).Count(&logs).Error)
		assert.Equal(t, oldLogs+logDelta, logs, name)
		count += delta
		assert.Equal(t, count, samples(), name)
	}
	send("HTTP filtered still logged", 502, 0, 1)
	body = `{"error":{"code":"ordinary","message":"ordinary"}}`
	send("ordinary counted", 502, 1, 1)
	body = `{"error":{"code":"throw","message":"ordinary"}}`
	send("throw defaults to counting", 502, 1, 1)
	body = `{"error":{"code":"invalid","message":"ordinary"}}`
	send("invalid defaults to counting", 502, 1, 1)
	body = `{"error":{"code":"cyber_policy","message":"blocked"}}`
	send("host policy still wins", 502, 0, 1)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"perf_metrics_setting.failure_filter_rules": `[{"id":"code","name":"code","enabled":true,"field":"error_code","mode":"exact","value":"ordinary"}]`}))
	body = `{"error":{"code":"ordinary","message":"ordinary"}}`
	send("admin filter still wins", 502, 0, 1)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"perf_metrics_setting.failure_filter_rules": "[]"}))
	status = 200
	body = `{"invalid":true}`
	send("parse filtered still logged", 502, 0, 1)
	body = `{"failed":true,"code":123}`
	send("business failure filtered", 502, 0, 0)
	body = `{"failed":true,"code":124}`
	send("business failure retained", 502, 1, 0)
	client := service.GetHttpClient()
	oldTransport := client.Transport
	t.Cleanup(func() { client.Transport = oldTransport })
	client.Transport = performanceFilterTransport{}
	send("transport filtered still logged", 500, 0, 1)
	client.Transport = oldTransport
	status = 502
	body = `{"error":{"code":"ignore","message":"ordinary"}}`
	nextSource := strings.Replace(source, `version: "1.0.0"`, `version: "1.0.1"`, 1)
	nextSource = strings.Replace(nextSource, taskPerformanceFilterHook, `export function shouldRecordPerformanceFailure(){return true;}`, 1)
	next, err := jsplugin.CompilePlugin(nextSource, jsplugin.Options{})
	require.NoError(t, err)
	duringRequest = func() {
		require.NoError(t, jsplugin.DefaultRegistry.ReplaceOverrides([]*jsplugin.LoadedPlugin{next}))
	}
	send("in flight stays pinned", 502, 0, 1)
	duringRequest = nil
	require.NoError(t, service.RefreshTaskPluginRoutes())
	status = 200
	body = `{"usage":{"input_tokens":486,"output_tokens":70}}`
	balance -= int64(common.QuotaPerUnit * 0.001)
	send("success unaffected", 200, 1, 0)
	var log model.Log
	require.NoError(t, db.Where("type = ?", model.LogTypeConsume).Order("id DESC").First(&log).Error)
	assert.Equal(t, 486, log.PromptTokens)
	assert.Equal(t, 70, log.CompletionTokens)
	// 钩子执行中取消请求：返回值即使为true，也不得采样。
	current, _ := jsplugin.DefaultRegistry.Generation().Get("performance-gate")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelSource := strings.Replace(source, taskPerformanceFilterHook, `export function shouldRecordPerformanceFailure(){console.log("cancel-filter");return true;}`, 1)
	cancelling, err := jsplugin.CompilePlugin(cancelSource, jsplugin.Options{Log: func(string) { cancel() }})
	require.NoError(t, err)
	originalEngine := current.Engine
	current.Engine = cancelling.Engine
	t.Cleanup(func() { current.Engine = originalEngine })
	req := httptest.NewRequest("POST", "/performance-gate", strings.NewReader(`{"model":"filter-alias"}`)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token.Key)
	status = 502
	body = `{"error":{"code":"ordinary"}}`
	engine.ServeHTTP(httptest.NewRecorder(), req)
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	assert.Equal(t, count, samples(), "取消不能生成失败样本")
	require.Eventually(t, func() bool { var u model.User; return db.First(&u, user.Id).Error == nil && u.Quota == balance }, 2*time.Second, 10*time.Millisecond)
}

type performanceFilterTransport struct{}

func (performanceFilterTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("simulated transport failure")
}
