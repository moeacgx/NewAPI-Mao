package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenGroupChannelRestrictsInitialSelectionAndRetry(t *testing.T) {
	for _, cached := range []bool{false, true} {
		t.Run(map[bool]string{true: "cache", false: "database"}[cached], func(t *testing.T) {
			db := setupChannelSelectAutoGroupsTest(t)
			createChannelSelectAutoGroupsChannel(t, db, 3101, "vip", "test-model", 1)
			createChannelSelectAutoGroupsChannel(t, db, 3102, "vip", "test-model", 10)
			createChannelSelectAutoGroupsChannel(t, db, 3103, "default", "test-model", 1)
			model.InitChannelCache()
			common.MemoryCacheEnabled = cached
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			common.SetContextKey(ctx, constant.ContextKeyTokenGroup, "vip,default")
			common.SetContextKey(ctx, constant.ContextKeyTokenGroups, []string{"vip", "default"})
			common.SetContextKey(ctx, constant.ContextKeyTokenGroupChannels, map[string]int{"vip": 3101, "default": 3103})
			param := &RetryParam{Ctx: ctx, TokenGroup: "vip,default", ModelName: "test-model", RequestPath: "/v1/chat/completions"}
			selected, group, err := CacheGetRandomSatisfiedChannel(param)
			require.NoError(t, err)
			require.NotNil(t, selected)
			assert.Equal(t, 3101, selected.Id, "不能被同组高优先级渠道覆盖")
			assert.Equal(t, "vip", group)
			param.ExcludeChannelID(3101, false)
			param.IncreaseRetry()
			selected, group, err = CacheGetRandomSatisfiedChannel(param)
			require.NoError(t, err)
			require.NotNil(t, selected)
			assert.Equal(t, 3103, selected.Id, "耗尽指定渠道应推进下一组")
			assert.Equal(t, "default", group)
		})
	}
}

func TestTokenGroupChannelIneligibleNeverSelectsAnotherChannel(t *testing.T) {
	for _, reason := range []string{"disabled", "wrong-model", "removed-group"} {
		t.Run(reason, func(t *testing.T) {
			db := setupChannelSelectAutoGroupsTest(t)
			createChannelSelectAutoGroupsChannel(t, db, 3301, "vip", "test-model", 0)
			group, modelName := "vip", "test-model"
			if reason == "wrong-model" {
				modelName = "other-model"
			}
			if reason == "removed-group" {
				group = "default"
			}
			createChannelSelectAutoGroupsChannel(t, db, 3302, group, modelName, 0)
			if reason == "disabled" {
				require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", 3302).Update("status", common.ChannelStatusManuallyDisabled).Error)
				require.NoError(t, model.UpdateAbilityStatus(3302, false))
			}
			model.InitChannelCache()
			for _, cached := range []bool{false, true} {
				common.MemoryCacheEnabled = cached
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
				common.SetContextKey(ctx, constant.ContextKeyTokenGroupChannels, map[string]int{"vip": 3302})
				selected, _, err := CacheGetRandomSatisfiedChannel(&RetryParam{Ctx: ctx, TokenGroup: "vip", ModelName: "test-model"})
				require.NoError(t, err)
				assert.Nil(t, selected, "缓存=%v，不能选到同组正常渠道", cached)
			}
		})
	}
}

func TestTokenGroupChannelMissingNeverFallsBackWithinGroup(t *testing.T) {
	db := setupChannelSelectAutoGroupsTest(t)
	createChannelSelectAutoGroupsChannel(t, db, 3201, "vip", "test-model", 0)
	model.InitChannelCache()
	for _, cached := range []bool{false, true} {
		common.MemoryCacheEnabled = cached
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Set("token_group_channels", map[string]int{"vip": 99999})
		param := &RetryParam{Ctx: ctx, TokenGroup: "vip", ModelName: "test-model"}
		selected, _, err := CacheGetRandomSatisfiedChannel(param)
		require.NoError(t, err)
		assert.Nil(t, selected)
		param.IncreaseRetry()
		param.RetryFallbackChannelIDs = map[int]struct{}{3201: {}}
		selected, _, err = CacheGetRandomSatisfiedChannel(param)
		require.NoError(t, err)
		assert.Nil(t, selected, "退让重试也必须保留限定")
	}
}
