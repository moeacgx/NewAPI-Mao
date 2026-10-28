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
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDistributeReselectionUsesActualGroupAndHonorsUserConcurrency(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(fmt.Sprintf("second_group_full_%t", blocked), func(t *testing.T) {
			originalDB, originalLogDB := model.DB, model.LOG_DB
			originalMemoryCache, originalRedis, originalRetry := common.MemoryCacheEnabled, common.RedisEnabled, common.RetryTimes
			dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
			db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Group{}))
			model.DB, model.LOG_DB = db, nil
			common.MemoryCacheEnabled, common.RedisEnabled, common.RetryTimes = true, false, 3
			t.Cleanup(func() {
				model.DB, model.LOG_DB = originalDB, originalLogDB
				common.MemoryCacheEnabled, common.RedisEnabled, common.RetryTimes = originalMemoryCache, originalRedis, originalRetry
				if originalMemoryCache && originalDB != nil && originalDB.Migrator().HasTable(&model.Channel{}) && originalDB.Migrator().HasTable(&model.Ability{}) {
					model.InitChannelCache()
				}
				sqlDB, err := db.DB()
				require.NoError(t, err)
				require.NoError(t, sqlDB.Close())
			})
			require.NoError(t, i18n.Init())

			const userID = 998100
			const modelName = "group-fallback-model"
			groups := []model.Group{
				{Id: 998101, Code: "fallback-first", Name: "首选分组", SingleUserConcurrencyLimit: 1},
				{Id: 998102, Code: "fallback-second", Name: "后备分组", SingleUserConcurrencyLimit: 1},
				{Id: 998103, Code: "fallback-third", Name: "末尾分组", SingleUserConcurrencyLimit: 1},
			}
			require.NoError(t, db.Create(&groups).Error)
			channels := make([]model.Channel, len(groups))
			for i, group := range groups {
				limit, priority, weight := 1, int64(100), uint(100)
				channels[i] = model.Channel{
					Id: group.Id, Type: constant.ChannelTypeOpenAI, Key: "test-key",
					Status: common.ChannelStatusEnabled, Name: group.Code,
					Group: group.Code, Models: modelName, Weight: &weight,
					Priority: &priority, ConcurrencyLimit: &limit,
				}
				if i == 0 {
					// 首选渠道选中后因密钥停用失败，确定性进入与抢租失败共用的重选链路。
					channels[i].ChannelInfo = model.ChannelInfo{
						IsMultiKey: true, MultiKeyMode: constant.MultiKeyModeRandom,
						MultiKeyStatusList: map[int]int{0: common.ChannelStatusAutoDisabled},
					}
				}
				require.NoError(t, db.Create(&channels[i]).Error)
				require.NoError(t, db.Create(&model.Ability{
					Group: group.Code, Model: modelName, ChannelId: group.Id,
					Enabled: true, Priority: &priority, Weight: weight,
				}).Error)
			}
			model.InitChannelCache()
			if blocked {
				lease, acquired := model.TryAcquireGroupUserConcurrency(userID, groups[1].Id, 1)
				require.True(t, acquired)
				t.Cleanup(lease.Release)
			}

			gin.SetMode(gin.TestMode)
			router := gin.New()
			var requestContext *gin.Context
			relayEntered := false
			authorizedGroups := []string{groups[0].Code, groups[1].Code, groups[2].Code}
			tokenGroup := strings.Join(authorizedGroups, ",")
			router.Use(func(c *gin.Context) {
				requestContext = c
				c.Set("id", userID)
				common.SetContextKey(c, constant.ContextKeyUsingGroup, tokenGroup)
				common.SetContextKey(c, constant.ContextKeyTokenGroup, tokenGroup)
				common.SetContextKey(c, constant.ContextKeyTokenGroups, authorizedGroups)
			})
			router.POST("/v1/chat/completions", Distribute(), func(c *gin.Context) {
				relayEntered = true
				assert.Equal(t, groups[1].Id, common.GetContextKeyInt(c, constant.ContextKeyChannelId))
				assert.Equal(t, groups[1].Code, common.GetContextKeyString(c, constant.ContextKeyUsingGroup))
				assert.Equal(t, groups[1].Code, common.GetContextKeyString(c, constant.ContextKeySelectedChannelGroup))
				assert.Equal(t, tokenGroup, common.GetContextKeyString(c, constant.ContextKeyTokenGroup))
				firstLease, firstAvailable := model.TryAcquireGroupUserConcurrency(userID, groups[0].Id, 1)
				assert.True(t, firstAvailable, "首选渠道失败后应释放首组用户槽位")
				if firstLease != nil {
					firstLease.Release()
				}
				secondLease, secondAvailable := model.TryAcquireGroupUserConcurrency(userID, groups[1].Id, 1)
				assert.False(t, secondAvailable, "实际后备组必须占用其用户槽位")
				if secondLease != nil {
					secondLease.Release()
				}
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"group-fallback-model"}`))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			require.NotNil(t, requestContext)
			if blocked {
				assert.Equal(t, http.StatusTooManyRequests, recorder.Code, recorder.Body.String())
				assert.False(t, relayEntered, "后备组用户并发满时不得绕到第三组")
				assert.Contains(t, recorder.Body.String(), `"code":"channel_concurrency_limit"`)
			} else {
				assert.Equal(t, http.StatusNoContent, recorder.Code, recorder.Body.String())
				assert.True(t, relayEntered)
			}
			assert.Equal(t, groups[1].Code, common.GetContextKeyString(requestContext, constant.ContextKeyUsingGroup))
			assert.Equal(t, groups[1].Code, common.GetContextKeyString(requestContext, constant.ContextKeySelectedChannelGroup))
			for i := range channels {
				assert.True(t, model.IsChannelConcurrencyAvailable(&channels[i]), "请求结束应释放渠道 %d 的槽位", channels[i].Id)
			}
			for i, group := range groups {
				lease, available := model.TryAcquireGroupUserConcurrency(userID, group.Id, 1)
				assert.Equal(t, !blocked || i != 1, available, "只应保留本测试预占的用户槽位")
				if lease != nil {
					lease.Release()
				}
			}
		})
	}
}
