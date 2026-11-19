package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDistributeModelCatalog(t *testing.T) {
	require.NoError(t, i18n.Init())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Model{}, &model.Vendor{}, &model.Channel{}, &model.Ability{}, &model.Group{}, &model.GroupAlias{}, &model.User{}, &model.Log{}))
	oldDB, oldLogDB := model.DB, model.LOG_DB
	oldMemory, oldRedis := common.MemoryCacheEnabled, common.RedisEnabled
	oldErrors, oldConsume := constant.ErrorLogEnabled, common.LogConsumeEnabled
	model.DB, model.LOG_DB = db, db
	common.MemoryCacheEnabled, common.RedisEnabled = true, false
	constant.ErrorLogEnabled, common.LogConsumeEnabled = true, true
	oldUsableGroups := setting.UserUsableGroups2JSONString()
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"public-catalog":"公开目录组"}`))
	t.Cleanup(func() {
		model.DB, model.LOG_DB = oldDB, oldLogDB
		common.MemoryCacheEnabled, common.RedisEnabled = oldMemory, oldRedis
		constant.ErrorLogEnabled, common.LogConsumeEnabled = oldErrors, oldConsume
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldUsableGroups))
		model.InvalidatePricingCache()
		require.NoError(t, sqlDB.Close())
	})
	group := model.Group{Code: "catalog-route", Name: "目录测试组", Status: model.GroupStatusActive}
	require.NoError(t, db.Create(&group).Error)
	user := model.User{Id: 722, Username: "catalog-user", Quota: 10000, Group: group.Code}
	require.NoError(t, db.Create(&user).Error)
	for _, entry := range []model.Model{
		{ModelName: "hidden-model"},
		{ModelName: "hidden-without-channel"},
		{ModelName: "no-channel", Status: 1},
		{ModelName: "catalog-prefix-", NameRule: model.NameRulePrefix},
		{ModelName: "-catalog-suffix", NameRule: model.NameRuleSuffix},
		{ModelName: "catalog-middle", NameRule: model.NameRuleContains},
	} {
		require.NoError(t, entry.Insert())
	}
	deleted := model.Model{ModelName: "deleted-model", Status: 1}
	require.NoError(t, deleted.Insert())
	require.NoError(t, db.Delete(&deleted).Error)
	channel := model.Channel{Id: 721, Name: "catalog-local", Type: constant.ChannelTypeOpenAI, Key: "local-only", Status: common.ChannelStatusEnabled, Group: group.Code,
		Models: "hidden-model,public-alias,gemini-2.5-flash-thinking-*,gpt-4o-gizmo-*,channel-only,private-ability"}
	require.NoError(t, db.Create(&channel).Error)
	for _, name := range []string{"hidden-model", "public-alias", "gemini-2.5-flash-thinking-*", "gpt-4o-gizmo-*"} {
		require.NoError(t, db.Create(&model.Ability{Group: group.Code, GroupId: group.Id, Model: name, ChannelId: channel.Id, Enabled: true}).Error)
	}
	for _, name := range []string{"public-alias", "gemini-2.5-flash-thinking-*", "gpt-4o-gizmo-*"} {
		require.NoError(t, db.Create(&model.Ability{Group: "public-catalog", Model: name, ChannelId: channel.Id, Enabled: true}).Error)
	}
	require.NoError(t, db.Create(&model.Ability{Group: group.Code, GroupId: group.Id, Model: "private-ability", ChannelId: channel.Id, Enabled: true}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: group.Code, GroupId: group.Id, Model: "channel-only", ChannelId: channel.Id, Enabled: false}).Error)
	model.InitChannelCache()
	var published []string
	for _, entry := range model.GetPricing() {
		published = append(published, entry.ModelName)
	}
	assert.NotContains(t, published, "hidden-model")
	assert.Contains(t, published, "public-alias")

	for _, test := range []struct {
		label, path, name, forced string
		status, errorLogs         int
	}{
		{label: "渠道有模型但目录未收录", name: "channel-only", status: 404},
		{label: "已删除元数据不收录", name: "deleted-model", status: 404},
		{label: "指定渠道也拒绝未收录", name: "channel-only", forced: "721", status: 404},
		{label: "启用能力仅在非公开分组", name: "private-ability", status: 404},
		{label: "路径模型未收录", path: "/v1beta/models/unlisted:generateContent", status: 404},
		{label: "实时模型在升级前拒绝", path: "/v1/realtime?model=unlisted", status: 404},
		{label: "空模型", status: 400},
		{label: "广场别名", name: "public-alias", status: 204},
		{label: "隐藏模型", name: "hidden-model", status: 204},
		{label: "Gemini thinking 路径", path: "/v1beta/models/gemini-2.5-flash-thinking-128:generateContent", status: 204},
		{label: "GPTS", name: "gpt-4o-gizmo-example", status: 204},
		{label: "目录收录但无渠道", name: "no-channel", status: 503, errorLogs: 1},
		{label: "隐藏模型无渠道仍记日志", name: "hidden-without-channel", status: 503, errorLogs: 1},
		{label: "前缀规则收录但无渠道", name: "catalog-prefix-member", status: 503, errorLogs: 1},
		{label: "后缀规则收录但无渠道", name: "member-catalog-suffix", status: 503, errorLogs: 1},
		{label: "包含规则收录但无渠道", name: "prefix-catalog-middle-suffix", status: 503, errorLogs: 1},
		{label: "历史视频查询无需选渠", path: "/v1/videos/task-id", status: 204},
	} {
		t.Run(test.label, func(t *testing.T) {
			engine := gin.New()
			engine.Use(BodyStorageCleanup(), func(c *gin.Context) {
				c.Set("id", user.Id)
				c.Set(common.RequestIdKey, t.Name())
				common.SetContextKey(c, constant.ContextKeyUsingGroup, group.Code)
				if test.forced != "" {
					common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, test.forced)
				}
			})
			entered := false
			terminal := func(c *gin.Context) {
				entered = true
				if c.Request.Method == http.MethodPost {
					assert.Equal(t, channel.Id, c.GetInt("channel_id"))
				}
				c.Status(http.StatusNoContent)
			}
			engine.POST("/v1/chat/completions", Distribute(), terminal)
			engine.POST("/v1beta/models/*action", Distribute(), terminal)
			engine.GET("/v1/realtime", AdmitRealtimeModel, PromptAuditRealtime(), Distribute(), terminal)
			engine.GET("/v1/videos/:task", Distribute(), terminal)
			path, method := test.path, http.MethodPost
			if path == "" {
				path = "/v1/chat/completions"
			} else if strings.HasPrefix(path, "/v1/") {
				method = http.MethodGet
			}
			req := httptest.NewRequest(method, path, strings.NewReader(fmt.Sprintf(`{"model":%q}`, test.name)))
			req.Header.Set("Content-Type", gin.MIMEJSON)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, req)
			assert.Equal(t, test.status, response.Code, response.Body.String())
			assert.Equal(t, test.status == 204, entered)
			if test.status == 404 {
				assert.Contains(t, response.Body.String(), `"code":"model_not_found"`)
			}
			var logs []model.Log
			require.NoError(t, db.Where("request_id = ?", t.Name()).Find(&logs).Error)
			require.Len(t, logs, test.errorLogs)
			for _, log := range logs {
				assert.Equal(t, model.LogTypeError, log.Type)
				assert.Zero(t, log.ChannelId)
				assert.Equal(t, test.name, log.ModelName)
			}
			var storedUser model.User
			require.NoError(t, db.First(&storedUser, user.Id).Error)
			assert.Equal(t, user.Quota, storedUser.Quota)
		})
	}
}

// 未收录模型必须在 PromptAudit 和后续中间件执行前被拒绝，且不留下日志。
func TestAdmitRequestModelRejectsBeforePromptAudit(t *testing.T) {
	require.NoError(t, i18n.Init())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Model{}, &model.Vendor{}, &model.Channel{}, &model.Ability{}, &model.Log{}))
	oldDB, oldLogDB := model.DB, model.LOG_DB
	model.DB, model.LOG_DB = db, db
	t.Cleanup(func() {
		model.DB, model.LOG_DB = oldDB, oldLogDB
		model.InvalidatePricingCache()
		require.NoError(t, sqlDB.Close())
	})
	model.InvalidatePricingCache()
	require.NoError(t, (&model.Model{ModelName: "listed-model", Status: 1}).Insert())

	for _, test := range []struct {
		name, method, path, body string
		wantStatus               int
	}{
		{name: "同步未收录", method: http.MethodPost, path: "/v1/chat/completions", body: `{"model":"unlisted"}`, wantStatus: http.StatusNotFound},
		{name: "Gemini 路径未收录", method: http.MethodPost, path: "/v1beta/models/unlisted:generateContent", body: `{}`, wantStatus: http.StatusNotFound},
		{name: "已收录", method: http.MethodPost, path: "/v1/chat/completions", body: `{"model":"listed-model"}`, wantStatus: http.StatusNoContent},
		{name: "空模型", method: http.MethodPost, path: "/v1/chat/completions", body: `{}`, wantStatus: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine := gin.New()
			auditCalls := 0
			distributeCalls := 0
			terminalCalls := 0
			engine.Use(BodyStorageCleanup())
			engine.POST("/v1/chat/completions",
				AdmitRequestModel(),
				func(c *gin.Context) { auditCalls++; c.Next() },
				func(c *gin.Context) { distributeCalls++; c.Next() },
				func(c *gin.Context) { terminalCalls++; c.Status(http.StatusNoContent) },
			)
			engine.POST("/v1beta/models/*action",
				AdmitRequestModel(),
				func(c *gin.Context) { auditCalls++; c.Next() },
				func(c *gin.Context) { terminalCalls++; c.Status(http.StatusNoContent) },
			)
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			req.Header.Set("Content-Type", gin.MIMEJSON)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, req)
			assert.Equal(t, test.wantStatus, response.Code, response.Body.String())
			if test.wantStatus == http.StatusNotFound {
				assert.Contains(t, response.Body.String(), `"code":"model_not_found"`)
				assert.Zero(t, auditCalls, "未收录模型不得进入 PromptAudit")
				assert.Zero(t, distributeCalls)
				assert.Zero(t, terminalCalls)
				var count int64
				require.NoError(t, db.Model(&model.Log{}).Count(&count).Error)
				assert.Zero(t, count, "未收录模型不得写入错误日志")
			}
		})
	}
}
