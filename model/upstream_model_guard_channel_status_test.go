package model

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpstreamModelGuardStatusUsesDatabaseOverStaleCache(t *testing.T) {
	for _, status := range []int{common.ChannelStatusAutoDisabled, common.ChannelStatusEnabled} {
		t.Run(map[int]string{common.ChannelStatusAutoDisabled: "迟到禁用", common.ChannelStatusEnabled: "迟到自动恢复"}[status], func(t *testing.T) {
			channel, _ := setupUpstreamModelGuardModelTest(t)
			common.MemoryCacheEnabled = true
			InitChannelCache()
			cached, err := CacheGetChannel(channel.Id)
			require.NoError(t, err)
			cached.ChannelInfo.MultiKeyPollingIndex = 0
			guardInfo := `{"upstream_model_guard":true,"status_reason":"guard mismatch"}`
			require.NoError(t, DB.Model(&Channel{}).Where("id = ?", channel.Id).Updates(map[string]any{"status": common.ChannelStatusManuallyDisabled, "other_info": guardInfo}).Error)
			require.NoError(t, DB.Model(&Ability{}).Where("channel_id = ?", channel.Id).Update("enabled", false).Error)
			require.Equal(t, common.ChannelStatusEnabled, cached.Status)

			assert.False(t, UpdateChannelStatusAutomatically(channel.Id, "first-key", status, ""))

			var stored Channel
			require.NoError(t, DB.First(&stored, channel.Id).Error)
			assert.Equal(t, common.ChannelStatusManuallyDisabled, stored.Status)
			assert.Equal(t, guardInfo, stored.OtherInfo)
			cached, err = CacheGetChannel(channel.Id)
			require.NoError(t, err)
			assert.Equal(t, common.ChannelStatusManuallyDisabled, cached.Status)
			assert.Equal(t, guardInfo, cached.OtherInfo)
			assert.Equal(t, 0, cached.ChannelInfo.MultiKeyPollingIndex, "拒绝旧请求不能回退本节点轮询游标")
		})
	}
}

func TestUpstreamModelGuardStatusAuthorityDoesNotDependOnReason(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	changed, err := ObserveUpstreamModelGuard(t.Context(), upstreamModelGuardTestRecord(channel, config), false)
	require.NoError(t, err)
	require.True(t, changed)
	assert.False(t, UpdateChannelStatusAutomatically(channel.Id, "", common.ChannelStatusEnabled, "manual operation"), "自动入口不能借助文案恢复渠道")
	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusManuallyDisabled, stored.Status)
	assert.Equal(t, true, stored.GetOtherInfo()["upstream_model_guard"])
	require.True(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusEnabled, ""), "人工入口无原因文本也应允许显式恢复")
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
	assert.NotEqual(t, true, stored.GetOtherInfo()["upstream_model_guard"])
}

func TestChannelStatusRollsBackCacheAndDatabaseWhenAbilityUpdateFails(t *testing.T) {
	channel, _ := setupUpstreamModelGuardModelTest(t)
	common.MemoryCacheEnabled = true
	InitChannelCache()
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register("guard-status:ability-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "abilities" {
			tx.AddError(errors.New("ability update rejected"))
		}
	}))
	assert.False(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusManuallyDisabled, "manual operation"))
	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
	assert.Equal(t, channel.OtherInfo, stored.OtherInfo)
	cached, err := CacheGetChannel(channel.Id)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, cached.Status)
	assert.Equal(t, channel.OtherInfo, cached.OtherInfo)
}
