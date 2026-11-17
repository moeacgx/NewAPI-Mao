package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSubscriptionDowngradeGroupTest(t *testing.T) (*Group, *Group) {
	t.Helper()
	defaultGroup, target := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&SubscriptionPlan{}, &UserSubscription{}))

	// 保存业务入口会刷新运行时投影，夹具在结束时恢复，避免影响其他测试。
	previousProjection := map[string]string{
		"GroupRatio":       ratio_setting.GroupRatio2JSONString(),
		"UserUsableGroups": setting.UserUsableGroups2JSONString(),
		"AutoGroups":       setting.AutoGroups2JsonString(),
	}
	common.OptionMapRWMutex.Lock()
	previousOptions := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	previousRedis := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		for key, value := range previousProjection {
			require.NoError(t, updateOptionMap(key, value))
		}
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptions
		common.OptionMapRWMutex.Unlock()
		common.RedisEnabled = previousRedis
	})
	return defaultGroup, target
}

func TestSubscriptionDowngradeReferencesBlockGroupDeletion(t *testing.T) {
	for _, entity := range []string{"plan", "subscription"} {
		for _, identifier := range []string{"code", "alias"} {
			t.Run(entity+"/"+identifier, func(t *testing.T) {
				_, target := setupSubscriptionDowngradeGroupTest(t)
				value := target.Code
				if identifier == "alias" {
					value = "legacy-downgrade"
					require.NoError(t, DB.Create(&GroupAlias{Alias: value, GroupId: target.Id}).Error)
				}
				if entity == "plan" {
					plan := &SubscriptionPlan{Title: "仅降级引用套餐", DowngradeGroup: value}
					require.NoError(t, DB.Create(plan).Error)
					// 禁用套餐仍然保留配置引用，不能因状态变化产生悬空目标。
					require.NoError(t, DB.Model(plan).Update("enabled", false).Error)
				} else {
					// 历史订阅的降级快照也属于业务引用。
					require.NoError(t, DB.Create(&UserSubscription{
						UserId: 77, DowngradeGroup: value, Status: "expired",
					}).Error)
				}

				err := SaveGroupConfig(nil, []int{target.Id})
				assert.EqualError(t, err, fmt.Sprintf("分组 %s 仍被非令牌业务数据引用，不能删除", target.Name))
				var retained Group
				require.NoError(t, DB.First(&retained, target.Id).Error)
				assert.Equal(t, target.Code, retained.Code)
				resolved, err := GetGroupByCodeOrAlias(value)
				require.NoError(t, err)
				assert.Equal(t, target.Id, resolved.Id)
			})
		}
	}
}

func TestSubscriptionDowngradeDeletionFailureRollsBackWholeSave(t *testing.T) {
	defaultGroup, unused := setupSubscriptionDowngradeGroupTest(t)
	target := &Group{Code: "downgrade-target", Name: "保留降级组", Ratio: 1, Status: GroupStatusActive}
	require.NoError(t, DB.Create(target).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		UserId: 88, Status: "active", DowngradeGroup: target.Code,
	}).Error)
	token := &Token{UserId: 88, Key: "downgrade-reference-rollback", Name: "待迁移令牌",
		Group: unused.Code, UnlimitedQuota: true}
	require.NoError(t, token.Insert())
	optionValue, err := common.Marshal(map[string]float64{"default": 1})
	require.NoError(t, err)
	option := &Option{Key: "TopupGroupRatio", Value: string(optionValue)}
	require.NoError(t, DB.Create(option).Error)

	// 无引用组先执行令牌迁移，后续遇到降级引用时，整个保存必须回滚。
	_, err = SaveGroupConfigWithOptionsAndResult(
		[]GroupConfig{{Id: defaultGroup.Id, Code: defaultGroup.Code, Name: "不应保存的改名",
			Ratio: defaultGroup.Ratio, Status: GroupStatusActive}},
		[]int{unused.Id, target.Id},
		map[string]string{"TopupGroupRatio": `{"default":2}`},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), target.Name)
	for _, original := range []*Group{defaultGroup, unused, target} {
		var stored Group
		require.NoError(t, DB.First(&stored, original.Id).Error)
		assert.Equal(t, original.Name, stored.Name)
		assert.Equal(t, original.Code, stored.Code)
	}
	var storedToken Token
	require.NoError(t, DB.First(&storedToken, token.Id).Error)
	assert.Equal(t, unused.Code, storedToken.Group)
	assert.Equal(t, TokenGroupModeExplicit, storedToken.GroupMode)
	var bindings []TokenGroupBinding
	require.NoError(t, DB.Where("token_id = ?", token.Id).Find(&bindings).Error)
	require.Len(t, bindings, 1)
	assert.Equal(t, unused.Id, bindings[0].GroupId)
	var storedOption Option
	require.NoError(t, DB.Where(&Option{Key: option.Key}).First(&storedOption).Error)
	assert.Equal(t, option.Value, storedOption.Value)
}

func TestSubscriptionDowngradeNonReferencesAllowGroupDeletion(t *testing.T) {
	for _, value := range []string{"", "unknown-group", "prefix-vip-suffix", "legacy-vip-suffix", "VIP", "current-code-owner"} {
		t.Run(value, func(t *testing.T) {
			_, target := setupSubscriptionDowngradeGroupTest(t)
			require.NoError(t, DB.Create(&GroupAlias{Alias: "legacy-vip", GroupId: target.Id}).Error)
			if value == "current-code-owner" {
				owner := &Group{Code: value, Name: "其他正式分组", Ratio: 1, Status: GroupStatusActive}
				require.NoError(t, DB.Create(owner).Error)
				require.NoError(t, DB.Create(&GroupAlias{Alias: value, GroupId: target.Id}).Error)
			}
			plan := &SubscriptionPlan{Title: "非目标引用", DowngradeGroup: value}
			sub := &UserSubscription{UserId: 99, Status: "active", DowngradeGroup: value}
			require.NoError(t, DB.Create(plan).Error)
			require.NoError(t, DB.Create(sub).Error)

			require.NoError(t, SaveGroupConfig(nil, []int{target.Id}))
			var deleted Group
			assert.ErrorIs(t, DB.First(&deleted, target.Id).Error, gorm.ErrRecordNotFound)
			var storedPlan SubscriptionPlan
			var storedSub UserSubscription
			require.NoError(t, DB.First(&storedPlan, plan.Id).Error)
			require.NoError(t, DB.First(&storedSub, sub.Id).Error)
			assert.Equal(t, value, storedPlan.DowngradeGroup)
			assert.Equal(t, value, storedSub.DowngradeGroup)
			if value == "current-code-owner" {
				owner, err := GetGroupByCodeOrAlias(value)
				require.NoError(t, err)
				assert.Equal(t, value, owner.Code)
			}
		})
	}
}

func TestSubscriptionDowngradeProtectedTargetRemainsValidAtExpiry(t *testing.T) {
	for _, identifier := range []string{"code", "alias"} {
		t.Run(identifier, func(t *testing.T) {
			defaultGroup, target := setupSubscriptionDowngradeGroupTest(t)
			value := target.Code
			if identifier == "alias" {
				value = "previous-downgrade-code"
				require.NoError(t, DB.Create(&GroupAlias{Alias: value, GroupId: target.Id}).Error)
			}
			user := &User{Username: "protected-downgrade", Group: defaultGroup.Code, Status: common.UserStatusEnabled}
			require.NoError(t, DB.Create(user).Error)
			now := GetDBTimestamp()
			sub := &UserSubscription{UserId: user.Id, Status: "active", StartTime: now - 200,
				EndTime: now - 100, UpgradeGroup: defaultGroup.Code, DowngradeGroup: value}
			require.NoError(t, DB.Create(sub).Error)

			require.NoError(t, SaveGroupConfig([]GroupConfig{{
				Id: target.Id, Code: target.Code, Name: "改名后的降级组",
				Ratio: target.Ratio, Status: GroupStatusActive,
			}}, nil))
			require.Error(t, SaveGroupConfig(nil, []int{target.Id}))
			count, err := ExpireDueSubscriptions(20)
			require.NoError(t, err)
			assert.Equal(t, 1, count)
			var storedUser User
			var storedSub UserSubscription
			require.NoError(t, DB.First(&storedUser, user.Id).Error)
			require.NoError(t, DB.First(&storedSub, sub.Id).Error)
			assert.Equal(t, value, storedUser.Group)
			assert.Equal(t, value, storedSub.DowngradeGroup)
			assert.Equal(t, "expired", storedSub.Status)
			resolved, err := GetGroupByCodeOrAlias(storedUser.Group)
			require.NoError(t, err)
			assert.Equal(t, target.Id, resolved.Id)
			assert.Equal(t, "改名后的降级组", resolved.Name)
		})
	}
}
