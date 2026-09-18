package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpstreamModelGuardRejectsLaterAutomaticStatusChanges(t *testing.T) {
	for _, multiKey := range []bool{false, true} {
		for _, automaticEnable := range []bool{false, true} {
			name := "单密钥自动禁用"
			if multiKey {
				name = "多密钥自动禁用"
			}
			if automaticEnable {
				name += "后启用"
			}
			t.Run(name, func(t *testing.T) {
				channel, info, _ := setupUpstreamModelGuardServiceTest(t)
				if multiKey {
					channel.Key = "first\nsecond"
					channel.ChannelInfo = model.ChannelInfo{IsMultiKey: true, MultiKeySize: 2, MultiKeyMode: constant.MultiKeyModePolling}
					require.NoError(t, model.DB.Model(channel).Updates(map[string]any{"key": channel.Key, "channel_info": channel.ChannelInfo}).Error)
				}
				BindUpstreamModelGuard(info)
				info.SetUpstreamResponseModelName("wrong-model")
				var disabled model.Channel
				require.NoError(t, model.DB.First(&disabled, channel.Id).Error)
				require.Equal(t, common.ChannelStatusManuallyDisabled, disabled.Status)
				if automaticEnable {
					EnableChannel(channel.Id, "first", channel.Name)
				} else {
					DisableChannel(*types.NewChannelError(channel.Id, channel.Type, channel.Name, multiKey, "first", true), "迟到的上游失败")
					if multiKey {
						DisableChannel(*types.NewChannelError(channel.Id, channel.Type, channel.Name, multiKey, "second", true), "另一个迟到失败")
					}
				}
				var stored model.Channel
				require.NoError(t, model.DB.First(&stored, channel.Id).Error)
				assert.Equal(t, common.ChannelStatusManuallyDisabled, stored.Status)
				assert.Equal(t, disabled.OtherInfo, stored.OtherInfo)
				assert.Equal(t, disabled.ChannelInfo, stored.ChannelInfo)
				var enabledAbilities int64
				require.NoError(t, model.DB.Model(&model.Ability{}).Where("channel_id = ? AND enabled = ?", channel.Id, true).Count(&enabledAbilities).Error)
				assert.Zero(t, enabledAbilities)
			})
		}
	}
}

func TestUpstreamModelGuardAllowsManualRecoveryAndLaterOrdinaryAutomation(t *testing.T) {
	channel, info, _ := setupUpstreamModelGuardServiceTest(t)
	BindUpstreamModelGuard(info)
	info.SetUpstreamResponseModelName("wrong-model")
	require.True(t, model.UpdateChannelStatus(channel.Id, "", common.ChannelStatusEnabled, "manual operation"))
	var stored model.Channel
	require.NoError(t, model.DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
	assert.NotEqual(t, true, stored.GetOtherInfo()["upstream_model_guard"])
	DisableChannel(*types.NewChannelError(channel.Id, channel.Type, channel.Name, false, channel.Key, true), "ordinary failure")
	require.NoError(t, model.DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusAutoDisabled, stored.Status)
	EnableChannel(channel.Id, channel.Key, channel.Name)
	require.NoError(t, model.DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
}

func TestUpstreamModelGuardTagRecoveryDoesNotBlockFutureAutoRecovery(t *testing.T) {
	channel, info, _ := setupUpstreamModelGuardServiceTest(t)
	channel.Tag = common.GetPointer("guard-recovery-tag")
	require.NoError(t, model.DB.Model(channel).Update("tag", channel.Tag).Error)
	BindUpstreamModelGuard(info)
	info.SetUpstreamResponseModelName("wrong-model")
	require.NoError(t, model.EnableChannelByTag(*channel.Tag))
	DisableChannel(*types.NewChannelError(channel.Id, channel.Type, channel.Name, false, channel.Key, true), "ordinary failure")
	EnableChannel(channel.Id, channel.Key, channel.Name)
	var stored model.Channel
	require.NoError(t, model.DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
	assert.NotEqual(t, true, stored.GetOtherInfo()["upstream_model_guard"])
}
