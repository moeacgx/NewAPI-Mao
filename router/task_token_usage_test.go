package router

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/controller"
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

// 自有最小插件夹具：同时上报两种 Token，预扣估算刻意不同于实际用量。
const tokenUsagePluginFixture = `
export const meta = {
  apiVersion: 1, key: "token-usage", name: "Token usage fixture", version: "1.0.0",
  author: {name:"Test",url:"https://example.com"},
  models: ["token-model"], auth: "api_key", fetchMode: "per_task",
  routes: [{method:"POST", path:"/token-usage", type:"submit", decode:"decode", render:"render", retainResult:false}],
  usageSchema: {
    input_tokens: {type:"number", unit:"token"},
    output_tokens: {type:"number", unit:"token"}
  },
  usageExamples: [{label:"Token usage", facts:{input_tokens:1000,output_tokens:70}}]
};
export function buildSubmitRequest(ctx) { return {url:ctx.baseUrl+"/run",method:"POST",body:ctx.requestBody}; }
export function parseSubmitResponse(ctx, response) {
  if (response.body.invalid) throw new Error("invalid provider response");
  if (response.body.pending) return {taskId:ctx.publicTaskId,taskData:response.body};
  if (response.body.failed) return {taskId:ctx.publicTaskId,taskData:response.body,immediate:{status:"FAILURE"}};
  return {taskId:ctx.publicTaskId,taskData:response.body,immediate:{status:"SUCCESS",progress:"100%"}};
}
export function extractUsage(ctx) { return ctx.usagePurpose==="billing_ratios" ? null : {input_tokens:32000}; }
export function extractUsageOnComplete(ctx, result, body) { return body.usage; }
export function buildQueryRequest() { throw new Error("同步测试不得轮询"); }
export function parseTaskResult() { return {status:"UNKNOWN"}; }
export const native = {
  decode(ctx) { return {kind:"submit",model:ctx.body.value.model,action:"run",requestBody:ctx.body.value}; },
  render(ctx, task) { return task.data; }
};`

func TestNativeTaskActualTokenLogs(t *testing.T) {
	for _, fixed := range []bool{true, false} {
		t.Run(fmt.Sprintf("fixed=%t", fixed), func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			require.NoError(t, i18n.Init())
			db := setupFeatureRouterAuthTest(t)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			model.InitDBColumns()
			require.NoError(t, db.AutoMigrate(&model.Option{}, &model.TaskPlugin{}, &model.Channel{}, &model.Token{}, &model.Log{}, &model.PerfMetric{}, &model.Group{}, &model.GroupAlias{}, &model.ChannelGroupBinding{}, &model.Ability{}, &model.PromptAuditConfig{}, &model.PromptAuditEndpoint{}, &model.RequestArchiveConfig{}, &model.RequestArchiveTarget{}, &model.PromptAuditQueueState{}, &model.RequestArchiveQueueState{}))
			oldRegistry, oldLogDB := jsplugin.DefaultRegistry, model.LOG_DB
			oldCache, oldBatch, oldLog := common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled
			oldQuota := common.QuotaPerUnit
			oldPrice := ratio_setting.ModelPrice2JSONString()
			jsplugin.DefaultRegistry, model.LOG_DB = jsplugin.NewRegistry(), db
			common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled = false, false, true
			common.QuotaPerUnit = 500000
			savedBilling := map[string]string{}
			oldPerformance := perf_metrics_setting.GetSetting()
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"perf_metrics_setting.enabled": "true", "perf_metrics_setting.failure_filter_rules": "[]"}))
			require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
				if strings.HasPrefix(key, "billing_setting.") {
					savedBilling[key] = value
				}
				return nil
			}))
			t.Cleanup(func() {
				jsplugin.DefaultRegistry, model.LOG_DB = oldRegistry, oldLogDB
				common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled = oldCache, oldBatch, oldLog
				common.QuotaPerUnit = oldQuota
				require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(oldPrice))
				require.NoError(t, config.GlobalConfig.LoadFromDB(savedBilling))
				oldRules, err := common.Marshal(oldPerformance.FailureFilterRules)
				require.NoError(t, err)
				require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"perf_metrics_setting.enabled": fmt.Sprint(oldPerformance.Enabled), "perf_metrics_setting.failure_filter_rules": string(oldRules)}))
			})
			mode := `{"token-model":"tiered_expr"}`
			if fixed {
				mode = `{}`
			}
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
				"billing_setting.billing_mode": mode,
				"billing_setting.billing_expr": `{"token-model":"u(\"input_tokens\") * 0.5 / 1000000"}`,
			}))
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"token-model":0.001}`))
			loaded, err := jsplugin.CompilePlugin(tokenUsagePluginFixture, jsplugin.Options{})
			require.NoError(t, err)
			require.NoError(t, db.Create(&model.TaskPlugin{Key: loaded.Meta.Key, Version: loaded.Meta.Version, APIVersion: 1, Source: tokenUsagePluginFixture, SourceHash: fmt.Sprintf("%x", sha256.Sum256([]byte(tokenUsagePluginFixture))), SourceKind: "custom", Active: true, Enabled: true}).Error)
			require.NoError(t, db.Create(&model.Option{Key: "TaskPluginEnabled", Value: "true"}).Error)
			group := model.Group{Code: "default", Name: "Default", Status: model.GroupStatusActive, UserSelectable: true, Ratio: 1}
			require.NoError(t, db.Create(&group).Error)
			user := model.User{Id: 95601, Username: "task-token-usage", Quota: 1000000, Status: common.UserStatusEnabled, Group: group.Code, Setting: `{"billing_preference":"wallet_only"}`}
			require.NoError(t, db.Create(&user).Error)
			token := model.Token{Id: 95601, UserId: user.Id, Key: "tasktokenusagetest", Status: common.TokenStatusEnabled, ExpiredTime: -1, RemainQuota: 1000000}
			require.NoError(t, db.Create(&token).Error)
			var upstreamBody string
			upstreamStatus := http.StatusOK
			upstreamCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upstreamCalls++
				assert.Equal(t, "/run", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(upstreamStatus)
				_, _ = io.WriteString(w, upstreamBody)
			}))
			t.Cleanup(server.Close)
			service.InitHttpClient()
			channel := model.Channel{Id: 95601, Type: constant.ChannelTypeTaskPlugin, Name: "token-usage", Key: "test", BaseURL: &server.URL, Models: "token-model", Group: group.Code, Status: common.ChannelStatusEnabled, Setting: common.GetPointer(`{"task_plugin_key":"token-usage"}`)}
			require.NoError(t, db.Create(&channel).Error)
			require.NoError(t, db.Create(&model.ChannelGroupBinding{ChannelId: channel.Id, GroupId: group.Id}).Error)
			require.NoError(t, db.Create(&model.Ability{ChannelId: channel.Id, Group: group.Code, GroupId: group.Id, Model: "token-model", Enabled: true}).Error)
			engine := gin.New()
			engine.GET("/api/perf-metrics/summary", controller.GetPerfMetricsSummary)
			engine.GET("/api/perf-metrics", controller.GetPerfMetrics)
			engine.NoRoute(SetPluginRouter(engine), func(c *gin.Context) { c.Status(http.StatusNotFound) })
			require.NoError(t, service.RefreshTaskPluginRoutes())
			balance := int64(1000000)
			before, err := perfmetrics.QuerySummaryAll(24, []string{group.Code})
			require.NoError(t, err)
			var baselineCount int64
			for _, metric := range before.Models {
				if metric.ModelName == "token-model" {
					baselineCount = metric.RequestCount
				}
			}
			assertCount := func(want int64) {
				t.Helper()
				summary, queryErr := perfmetrics.QuerySummaryAll(24, []string{group.Code})
				require.NoError(t, queryErr)
				var count int64
				for _, metric := range summary.Models {
					if metric.ModelName == "token-model" {
						count = metric.RequestCount
					}
				}
				assert.Equal(t, baselineCount+want, count)
			}
			for _, tc := range []struct {
				input, loggedOutput, charge int
				output                      float64
			}{{486, 70, 122, 70}, {1000, 0, 250, 0}, {0, 0, 0, 0}, {486, 0, 122, 2147483648}} {
				upstreamBody = fmt.Sprintf(`{"usage":{"input_tokens":%d,"output_tokens":%.0f}}`, tc.input, tc.output)
				request := httptest.NewRequest(http.MethodPost, "/token-usage", strings.NewReader(`{"model":"token-model"}`))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", "Bearer "+token.Key)
				recorder := httptest.NewRecorder()
				engine.ServeHTTP(recorder, request)
				require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
				assert.JSONEq(t, upstreamBody, recorder.Body.String())
				var log model.Log
				require.NoError(t, db.Where("type = ?", model.LogTypeConsume).Order("id DESC").First(&log).Error)
				assert.Equal(t, tc.input, log.PromptTokens)
				assert.Equal(t, tc.loggedOutput, log.CompletionTokens)
				charge := tc.charge
				if fixed {
					charge = 500
				}
				assert.Equal(t, charge, log.Quota)
				balance -= int64(charge)
				var actualUser model.User
				var actualToken model.Token
				require.NoError(t, db.First(&actualUser, user.Id).Error)
				require.NoError(t, db.First(&actualToken, token.Id).Error)
				assert.Equal(t, balance, actualUser.Quota)
				assert.EqualValues(t, balance, actualToken.RemainQuota)
			}
			assertCount(4)
			metrics, err := perfmetrics.Query(perfmetrics.QueryParams{Model: "token-model", Group: group.Code, Hours: 24})
			require.NoError(t, err)
			require.Len(t, metrics.Groups, 1)
			assert.Zero(t, metrics.Groups[0].AvgTtftMs)
			send := func(body string) *httptest.ResponseRecorder {
				request := httptest.NewRequest(http.MethodPost, "/token-usage", strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", "Bearer "+token.Key)
				recorder := httptest.NewRecorder()
				engine.ServeHTTP(recorder, request)
				if recorder.Code >= 400 {
					require.Eventually(t, func() bool {
						var charged model.User
						return db.First(&charged, user.Id).Error == nil && charged.Quota == balance
					}, 2*time.Second, 10*time.Millisecond)
				}
				return recorder
			}
			upstreamStatus = http.StatusBadGateway
			assert.Equal(t, http.StatusBadGateway, send(`{"model":"token-model"}`).Code)
			assertCount(5)
			// 上游结构化策略错误不能污染模型广场失败率。
			upstreamBody = `{"error":{"code":"cyber_policy","message":"blocked"}}`
			assert.Equal(t, http.StatusBadGateway, send(`{"model":"token-model"}`).Code)
			assertCount(5)
			upstreamStatus = http.StatusOK
			upstreamBody = `{"invalid":true}`
			assert.Equal(t, http.StatusBadGateway, send(`{"model":"token-model"}`).Code)
			assertCount(6)
			// 配置过滤排除上游状态错误，但不应影响计费失败退款链。
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"perf_metrics_setting.failure_filter_rules": `[{"id":"task-http","name":"task-http","enabled":true,"field":"error_code","mode":"exact","value":"fail_to_fetch_task"}]`}))
			upstreamStatus = http.StatusBadGateway
			assert.Equal(t, http.StatusBadGateway, send(`{"model":"token-model"}`).Code)
			assertCount(6)
			callsBefore := upstreamCalls
			assert.Equal(t, http.StatusBadRequest, send(`{}`).Code)
			assert.Equal(t, callsBefore, upstreamCalls)
			assertCount(6)
			// 未发供应商请求的余额不足不能污染模型可靠性。
			require.NoError(t, db.Model(&model.User{}).Where("id = ?", user.Id).Update("quota", 0).Error)
			balance = 0
			assert.Equal(t, http.StatusForbidden, send(`{"model":"token-model"}`).Code)
			assert.Equal(t, callsBefore, upstreamCalls)
			assertCount(6)
			balance = 1000000
			require.NoError(t, db.Model(&model.User{}).Where("id = ?", user.Id).Update("quota", balance).Error)
			upstreamStatus = http.StatusOK
			upstreamBody = `{"failed":true}`
			assert.Equal(t, http.StatusBadGateway, send(`{"model":"token-model"}`).Code)
			assertCount(7)
			upstreamBody = `{"usage":{"input_tokens":486,"output_tokens":70}}`
			// 客户端取消即便发生于供应商调用阶段也不计入失败率。
			client := service.GetHttpClient()
			previousTransport := client.Transport
			cancelCtx, cancel := context.WithCancel(context.Background())
			client.Transport = taskCancelTransport{cancel: cancel}
			cancelRequest := httptest.NewRequest(http.MethodPost, "/token-usage", strings.NewReader(`{"model":"token-model"}`)).WithContext(cancelCtx)
			cancelRequest.Header.Set("Content-Type", "application/json")
			cancelRequest.Header.Set("Authorization", "Bearer "+token.Key)
			engine.ServeHTTP(httptest.NewRecorder(), cancelRequest)
			client.Transport = previousTransport
			cancel()
			require.Eventually(t, func() bool {
				var charged model.User
				return db.First(&charged, user.Id).Error == nil && charged.Quota == balance
			}, 2*time.Second, 10*time.Millisecond)
			assertCount(7)
			// 广场的两个公开查询面都必须能读取同步任务样本。
			for _, path := range []string{"/api/perf-metrics/summary?hours=24", "/api/perf-metrics?model=token-model&hours=24"} {
				response := httptest.NewRecorder()
				engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
				require.Equal(t, http.StatusOK, response.Code)
				assert.Contains(t, response.Body.String(), `"model_name":"token-model"`)
				assert.Contains(t, response.Body.String(), `"success_rate":`)
			}
			if fixed {
				// retainResult:false 下偶然返回异步任务也不能把提交成功算成推理成功。
				upstreamBody = `{"pending":true}`
				assert.Equal(t, http.StatusOK, send(`{"model":"token-model"}`).Code)
				assertCount(7)
			}
		})
	}
}

type taskCancelTransport struct{ cancel context.CancelFunc }

func (transport taskCancelTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.cancel()
	return nil, request.Context().Err()
}
