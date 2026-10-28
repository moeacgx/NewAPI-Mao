package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheGetRandomSatisfiedChannelConcurrencyFallback(t *testing.T) {
	for _, cached := range []bool{true, false} {
		for _, mode := range []string{"explicit", "auto"} {
			for _, tc := range []struct {
				name           string
				groups         []string
				defaultExists  bool
				defaultFull    bool
				restrictVIP    bool
				excludeDefault bool
				wantChannel    int
				wantEmpty      bool
			}{
				{name: "later group available", groups: []string{"vip", "default"}, defaultExists: true, wantChannel: 2402},
				{name: "skip empty middle group", groups: []string{"vip", "empty", "default"}, defaultExists: true, wantChannel: 2402},
				{name: "all groups full", groups: []string{"vip", "default"}, defaultExists: true, defaultFull: true},
				{name: "full then missing model", groups: []string{"vip", "default"}},
				{name: "missing model then full", groups: []string{"default", "vip"}},
				{name: "keep required channel", groups: []string{"vip", "default"}, defaultExists: true, restrictVIP: true, wantChannel: 2402},
				{name: "keep channel exclusions", groups: []string{"vip", "default"}, defaultExists: true, excludeDefault: true},
				{name: "all groups missing model", groups: []string{"empty", "default"}, wantEmpty: true},
			} {
				cacheName := "database"
				if cached {
					cacheName = "cache"
				}
				t.Run(cacheName+"/"+mode+"/"+tc.name, func(t *testing.T) {
					db := setupChannelSelectAutoGroupsTest(t)
					oldRedis := common.RedisEnabled
					common.RedisEnabled = false
					t.Cleanup(func() { common.RedisEnabled = oldRedis })
					common.MemoryCacheEnabled = cached
					require.NoError(t, setting.UpdateMaxTokenAutoGroups("3"))
					require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"默认","vip":"优先","empty":"空组"}`))
					require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":2,"empty":1}`))
					const modelName = "concurrency-fallback-model"
					createChannelSelectAutoGroupsChannel(t, db, 2401, "vip", modelName, 100)
					if tc.defaultExists {
						createChannelSelectAutoGroupsChannel(t, db, 2402, "default", modelName, 100)
						createChannelSelectAutoGroupsChannel(t, db, 2403, "default", modelName, 50)
					}
					// 即使其他组有空闲渠道，也只能在请求的授权列表中选取。
					createChannelSelectAutoGroupsChannel(t, db, 2404, "unauthorized", modelName, 100)
					if tc.restrictVIP {
						createChannelSelectAutoGroupsChannel(t, db, 2405, "vip", modelName, 100)
					}
					fullIDs := []int{2401}
					if tc.defaultFull {
						fullIDs = append(fullIDs, 2402, 2403)
					}
					for _, id := range fullIDs {
						require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", id).Update("concurrency_limit", 1).Error)
						channel := &model.Channel{Id: id, ConcurrencyLimit: common.GetPointer(1)}
						lease, acquired := model.TryAcquireChannelConcurrencyLease(channel)
						require.True(t, acquired)
						t.Cleanup(func() { model.ReleaseChannelConcurrencyLease(lease) })
					}
					model.InitChannelCache()

					ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
					groups := tc.groups
					tokenGroup := strings.Join(groups, ",")
					if mode == "auto" {
						tokenGroup = "auto"
						common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
						common.SetContextKey(ctx, constant.ContextKeyTokenAutoGroups, groups)
					} else {
						common.SetContextKey(ctx, constant.ContextKeyTokenGroup, tokenGroup)
						common.SetContextKey(ctx, constant.ContextKeyTokenGroups, groups)
					}
					if tc.restrictVIP {
						common.SetContextKey(ctx, constant.ContextKeyTokenGroupChannels, map[string]int{"vip": 2401})
					}
					param := &RetryParam{Ctx: ctx, TokenGroup: tokenGroup, ModelName: modelName,
						RequestPath: "/v1/chat/completions", Retry: common.GetPointer(0)}
					if tc.excludeDefault {
						param.ExcludeChannelID(2402, true)
						param.ExcludeChannelID(2403, true)
						param.SetRetry(1)
					}
					channel, group, err := CacheGetRandomSatisfiedChannel(param)
					if tc.wantEmpty {
						require.NoError(t, err)
						assert.Nil(t, channel)
						return
					}
					if tc.wantChannel == 0 {
						require.ErrorIs(t, err, model.ErrChannelConcurrencyLimitReached)
						assert.Nil(t, channel)
						assert.Equal(t, "vip", group)
						return
					}
					require.NoError(t, err)
					require.NotNil(t, channel)
					assert.Equal(t, tc.wantChannel, channel.Id)
					assert.Equal(t, "default", group)
					assert.Equal(t, "default", common.GetContextKeyString(ctx, constant.ContextKeyAutoGroup))
					assert.Equal(t, len(groups)-1, retryGroupIndex(ctx))
					assert.Equal(t, 0, retryGroupStartIndex(ctx))
					assert.Equal(t, tokenGroup, param.TokenGroup)

					// 切换分组后继续本组重试，不重新访问已满的前组。
					param.ExcludeChannelID(channel.Id, true)
					param.IncreaseRetry()
					channel, group, err = CacheGetRandomSatisfiedChannel(param)
					require.NoError(t, err)
					require.NotNil(t, channel)
					assert.Equal(t, 2403, channel.Id)
					assert.Equal(t, "default", group)
				})
			}
		}
	}
}

func TestCacheGetRandomSatisfiedChannelSingleGroupConcurrencyDoesNotExpandScope(t *testing.T) {
	db := setupChannelSelectAutoGroupsTest(t)
	oldRedis := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = oldRedis })
	const modelName = "single-group-full-model"
	createChannelSelectAutoGroupsChannel(t, db, 2411, "vip", modelName, 0)
	createChannelSelectAutoGroupsChannel(t, db, 2412, "default", modelName, 0)
	require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", 2411).Update("concurrency_limit", 1).Error)
	lease, acquired := model.TryAcquireChannelConcurrencyLease(&model.Channel{Id: 2411, ConcurrencyLimit: common.GetPointer(1)})
	require.True(t, acquired)
	t.Cleanup(func() { model.ReleaseChannelConcurrencyLease(lease) })
	model.InitChannelCache()
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyTokenGroup, "vip")
	common.SetContextKey(ctx, constant.ContextKeyTokenGroups, []string{"vip"})
	channel, group, err := CacheGetRandomSatisfiedChannel(&RetryParam{Ctx: ctx, TokenGroup: "vip", ModelName: modelName})
	require.ErrorIs(t, err, model.ErrChannelConcurrencyLimitReached)
	assert.Nil(t, channel)
	assert.Equal(t, "vip", group)
}
