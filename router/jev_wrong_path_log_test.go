package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/perf_metrics_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const jevWrongPathFixture = `
export const meta = {
 apiVersion:1,key:"cloudflare-jev",name:"Cloudflare Jev",version:"1.0.0",
 author:{name:"Test"},models:["typesafe/jev"],fetchMode:"per_task",
 routes:[{method:"POST",path:"/v1/systemone",type:"submit",decode:"decode",render:"render",retainResult:false}]
};
export function buildSubmitRequest(){return {};}
export function parseSubmitResponse(){return {};}
export function buildQueryRequest(){return {};}
export function parseTaskResult(){return {};}
export const native={decode:function(ctx){return {kind:"submit",model:ctx.body.value.model,requestBody:ctx.body.value};},render:function(ctx,task){return task;}};
`

func TestCloudflareJevWrongPathKeepsLogsWithoutPerformanceSamples(t *testing.T) {
	for _, memory := range []bool{false, true} {
		t.Run(fmt.Sprint(memory), func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			require.NoError(t, i18n.Init())
			db := setupFeatureRouterAuthTest(t)
			model.InitDBColumns()
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			require.NoError(t, db.AutoMigrate(&model.Log{}, &model.PerfMetric{}, &model.Channel{}, &model.Ability{}, &model.Group{}, &model.GroupAlias{}, &model.ChannelGroupBinding{}))
			oldLog, oldCache, oldErrors, oldRegistry := model.LOG_DB, common.MemoryCacheEnabled, constant.ErrorLogEnabled, jsplugin.DefaultRegistry
			oldPerformance := perf_metrics_setting.GetSetting()
			oldGroupRatios := ratio_setting.GroupRatio2JSONString()
			require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"jev-log-group":1}`))
			require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"perf_metrics_setting.enabled": "true"}))
			model.LOG_DB = db
			common.MemoryCacheEnabled = memory
			constant.ErrorLogEnabled = true
			jsplugin.DefaultRegistry = jsplugin.NewRegistry()
			_, err = jsplugin.DefaultRegistry.RegisterFactory(jevWrongPathFixture, jsplugin.Options{})
			require.NoError(t, err)
			t.Cleanup(func() {
				model.LOG_DB = oldLog
				common.MemoryCacheEnabled = oldCache
				constant.ErrorLogEnabled = oldErrors
				jsplugin.DefaultRegistry = oldRegistry
				require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{"perf_metrics_setting.enabled": fmt.Sprint(oldPerformance.Enabled)}))
				require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroupRatios))
			})
			group := model.Group{Code: "jev-log-group", Name: "决策测试", Status: model.GroupStatusActive, Ratio: 1, UserSelectable: true}
			require.NoError(t, db.Create(&group).Error)
			user := model.User{Id: 98601, Username: "jev-log-user", Quota: 100000, Status: common.UserStatusEnabled, Group: group.Code}
			require.NoError(t, db.Create(&user).Error)
			channel := model.Channel{Id: 98601, Type: constant.ChannelTypeTaskPlugin, Name: "decision", Key: "test", Models: "decision-alias", Group: group.Code, Status: common.ChannelStatusEnabled, Setting: common.GetPointer(`{"task_plugin_key":"cloudflare-jev"}`)}
			require.NoError(t, db.Create(&channel).Error)
			require.NoError(t, db.Create(&model.ChannelGroupBinding{ChannelId: channel.Id, GroupId: group.Id}).Error)
			require.NoError(t, db.Create(&model.Ability{ChannelId: channel.Id, Group: group.Code, GroupId: group.Id, Model: "decision-alias", Enabled: true}).Error)
			if memory {
				model.InitChannelCache()
			}
			usingGroup := group.Code
			forbidModel := false
			var tokenGroups []string
			engine := gin.New()
			engine.Use(func(c *gin.Context) {
				c.Set("id", user.Id)
				c.Set("username", user.Username)
				c.Set(common.RequestIdKey, c.GetHeader("X-Request-ID"))
				common.SetContextKey(c, constant.ContextKeyUserGroup, group.Code)
				common.SetContextKey(c, constant.ContextKeyUsingGroup, usingGroup)
				common.SetContextKey(c, constant.ContextKeyTokenGroup, usingGroup)
				common.SetContextKey(c, constant.ContextKeyTokenGroups, tokenGroups)
				common.SetContextKey(c, constant.ContextKeyTokenAutoGroups, []string{group.Code})
				if forbidModel {
					common.SetContextKey(c, constant.ContextKeyTokenModelLimitEnabled, true)
					common.SetContextKey(c, constant.ContextKeyTokenModelLimit, map[string]bool{"other": true})
				}
				if forced := c.GetHeader("X-Forced-Channel"); forced != "" {
					common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, forced)
				}
				c.Next()
			})
			called := 0
			for _, path := range []string{"/v1/responses", "/v1/chat/completions", "/v1/systemone", "/v1/task/plugins/cloudflare-jev"} {
				engine.POST(path, middleware.Distribute(), func(c *gin.Context) { called++; c.Status(http.StatusNoContent) })
			}
			before, err := perfmetrics.QuerySummaryAll(24, []string{group.Code})
			require.NoError(t, err)
			testRequest := func(name, path, modelName, forced string, wantStatus int, wantLogs int64) {
				t.Helper()
				var beforeLogs int64
				require.NoError(t, db.Model(&model.Log{}).Count(&beforeLogs).Error)
				req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(fmt.Sprintf(`{"model":%q}`, modelName)))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Request-ID", name)
				req.Header.Set("X-Forced-Channel", forced)
				rec := httptest.NewRecorder()
				engine.ServeHTTP(rec, req)
				assert.Equal(t, wantStatus, rec.Code, rec.Body.String())
				if wantStatus >= 400 {
					assert.Contains(t, rec.Body.String(), `"error"`)
				}
				var count int64
				require.NoError(t, db.Model(&model.Log{}).Count(&count).Error)
				assert.Equal(t, beforeLogs+wantLogs, count, name)
			}
			testRequest("responses", "/v1/responses", "decision-alias", "", 503, 1)
			testRequest("chat", "/v1/chat/completions", "decision-alias", "", 503, 1)
			testRequest("forced", "/v1/responses", "decision-alias", "98601", 503, 1)
			assert.Zero(t, called)
			usingGroup = "empty," + group.Code
			tokenGroups = []string{"empty", group.Code}
			testRequest("ordered", "/v1/responses", "decision-alias", "", 503, 1)
			usingGroup = "auto"
			testRequest("auto", "/v1/responses", "decision-alias", "", 503, 1)
			usingGroup = group.Code
			tokenGroups = nil
			testRequest("unknown-model", "/v1/responses", "unknown-alias", "", 503, 1)
			usingGroup = "unrelated"
			testRequest("other-group", "/v1/responses", "decision-alias", "", 503, 1)
			usingGroup = group.Code
			forbidModel = true
			testRequest("forbidden", "/v1/responses", "decision-alias", "", 403, 1)
			forbidModel = false
			jsplugin.DefaultRegistry.SetEnabled(false)
			testRequest("disabled-plugin", "/v1/responses", "decision-alias", "", 503, 1)
			jsplugin.DefaultRegistry.SetEnabled(true)
			brokenKeys := model.ChannelInfo{IsMultiKey: true, MultiKeyStatusList: map[int]int{0: common.ChannelStatusManuallyDisabled}}
			require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("channel_info", brokenKeys).Error)
			if memory {
				model.InitChannelCache()
			}
			testRequest("valid-native-failure", "/v1/systemone", "decision-alias", "", 503, 1)
			testRequest("valid-generic-failure", "/v1/task/plugins/cloudflare-jev", "decision-alias", "", 503, 1)
			require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("setting", `{"task_plugin_key":"other-plugin"}`).Error)
			if memory {
				model.InitChannelCache()
			}
			testRequest("other-plugin", "/v1/responses", "decision-alias", "", 503, 1)
			require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("setting", `{"task_plugin_key":"cloudflare-jev"}`).Error)
			other := model.Channel{Id: 98602, Type: constant.ChannelTypeOpenAI, Name: "same-alias", Key: "test", Models: "decision-alias", Group: group.Code, Status: common.ChannelStatusEnabled, ChannelInfo: brokenKeys}
			require.NoError(t, db.Create(&other).Error)
			require.NoError(t, db.Create(&model.ChannelGroupBinding{ChannelId: other.Id, GroupId: group.Id}).Error)
			require.NoError(t, db.Create(&model.Ability{ChannelId: other.Id, Group: group.Code, GroupId: group.Id, Model: "decision-alias", Enabled: true}).Error)
			if memory {
				model.InitChannelCache()
			}
			testRequest("mixed-provider-failure", "/v1/responses", "decision-alias", "", 503, 1)
			testRequest("mixed-forced-failure", "/v1/responses", "decision-alias", "98601", 503, 1)
			require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", other.Id).Update("status", common.ChannelStatusManuallyDisabled).Error)
			require.NoError(t, db.Model(&model.Ability{}).Where("channel_id = ?", other.Id).Update("enabled", false).Error)
			require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channel.Id).Updates(map[string]any{"concurrency_limit": 1, "channel_info": model.ChannelInfo{}}).Error)
			require.NoError(t, db.First(&channel, channel.Id).Error)
			if memory {
				model.InitChannelCache()
			}
			lease, acquired := model.TryAcquireChannelConcurrencyLease(&channel)
			require.True(t, acquired)
			testRequest("valid-concurrency", "/v1/systemone", "decision-alias", "", 503, 1)
			assert.True(t, model.ReleaseChannelConcurrencyLease(lease))
			require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channel.Id).Updates(map[string]any{"type": constant.ChannelTypeOpenAI, "channel_info": model.ChannelInfo{}}).Error)
			if memory {
				model.InitChannelCache()
			}
			testRequest("ordinary-alias", "/v1/responses", "decision-alias", "", http.StatusNoContent, 0)
			assert.Equal(t, 1, called)
			after, err := perfmetrics.QuerySummaryAll(24, []string{group.Code})
			require.NoError(t, err)
			assert.Equal(t, before, after, "分发拒绝和错误日志不产生模型性能样本")
			var actual model.User
			require.NoError(t, db.First(&actual, user.Id).Error)
			assert.EqualValues(t, 100000, actual.Quota)
		})
	}
}
