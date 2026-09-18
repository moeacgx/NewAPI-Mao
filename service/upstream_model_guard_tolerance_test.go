package service

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpstreamModelGuardCountsRequestsAndResetsAcrossRules(t *testing.T) {
	channel, info, config := setupUpstreamModelGuardServiceTest(t)
	config.Rules = append(config.Rules, UpstreamModelGuardRule{Enabled: true, GroupCodes: []string{"other"}, Model: "client-other", UpstreamModels: []string{"allowed"}})
	_, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules, FailureThreshold: common.GetPointer(2)}, 42)
	require.NoError(t, err)
	for index, request := range []struct {
		group, requested string
		models           []string
		wantDisabled     bool
		recordCount      int64
	}{
		{"default", "client", []string{"wrong", "wrong", "provider"}, false, 1},
		{"default", "client", []string{""}, false, 1},
		{"other", "client-other", []string{"allowed", "allowed"}, false, 1},
		{"default", "client", []string{"provider", "wrong", "wrong-again"}, false, 2},
		{"other", "client-other", []string{"wrong"}, true, 3},
	} {
		info.RequestId = fmt.Sprintf("request-%d", index)
		info.UsingGroup, info.OriginModelName = request.group, request.requested
		finish := BindUpstreamModelGuard(info)
		for _, actual := range request.models {
			info.SetUpstreamResponseModelName(actual)
		}
		finish()
		finish()
		enabled, err := model.IsChannelEnabledAfterUpstreamModelGuard(t.Context(), channel.Id)
		require.NoError(t, err)
		assert.Equal(t, request.wantDisabled, !enabled)
		_, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
		require.NoError(t, err)
		assert.Equal(t, request.recordCount, total)
	}
}

func TestUpstreamModelGuardWhitelistSkipsWholeChannelAndPersistsOnLegacySave(t *testing.T) {
	channel, info, config := setupUpstreamModelGuardServiceTest(t)
	excluded := []int{channel.Id, channel.Id}
	updated, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules, ExcludedChannelIDs: &excluded, FailureThreshold: common.GetPointer(2)}, 42)
	require.NoError(t, err)
	assert.Equal(t, []int{channel.Id}, updated.ExcludedChannelIDs)
	assert.Equal(t, channel.Name, updated.ExcludedChannels[0].Name)
	updated, err = SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: updated.ConfigVersion, Enabled: true, Rules: config.Rules}, 42)
	require.NoError(t, err)
	assert.Equal(t, 2, updated.FailureThreshold)
	assert.Equal(t, []int{channel.Id}, updated.ExcludedChannelIDs)
	for index := 0; index < 3; index++ {
		info.RequestId = fmt.Sprintf("excluded-%d", index)
		finish := BindUpstreamModelGuard(info)
		info.SetUpstreamResponseModelName("wrong")
		finish()
	}
	enabled, err := model.IsChannelEnabledAfterUpstreamModelGuard(t.Context(), channel.Id)
	require.NoError(t, err)
	assert.True(t, enabled)
	_, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	assert.Zero(t, total)
}

func TestUpstreamModelGuardToleranceRejectsInvalidConfiguration(t *testing.T) {
	_, _, config := setupUpstreamModelGuardServiceTest(t)
	for _, threshold := range []int{0, -1, 101} {
		_, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Rules: config.Rules, FailureThreshold: &threshold}, 42)
		require.Error(t, err)
	}
	for _, ids := range [][]int{{0}, {-1}, {999999}} {
		_, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Rules: config.Rules, ExcludedChannelIDs: &ids}, 42)
		require.Error(t, err)
	}
}

func TestUpstreamModelGuardDeletedExcludedChannelCanBeRemoved(t *testing.T) {
	channel, _, config := setupUpstreamModelGuardServiceTest(t)
	ids := []int{channel.Id}
	config, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Rules: config.Rules, ExcludedChannelIDs: &ids}, 42)
	require.NoError(t, err)
	require.NoError(t, model.DB.Delete(channel).Error)
	loaded, err := GetUpstreamModelGuardConfig(t.Context())
	require.NoError(t, err)
	assert.Equal(t, ids, loaded.ExcludedChannelIDs)
	assert.Empty(t, loaded.ExcludedChannels)
	config, err = SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Rules: config.Rules, ExcludedChannelIDs: &ids}, 42)
	require.NoError(t, err)
	ids = []int{}
	config, err = SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Rules: config.Rules, ExcludedChannelIDs: &ids}, 42)
	require.NoError(t, err)
	assert.Empty(t, config.ExcludedChannelIDs)
}

func TestUpstreamModelGuardRequestDedupSurvivesNewBinding(t *testing.T) {
	channel, info, config := setupUpstreamModelGuardServiceTest(t)
	_, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules, FailureThreshold: common.GetPointer(2)}, 42)
	require.NoError(t, err)
	for index := 0; index < 2; index++ {
		copyInfo := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelId: channel.Id}, RequestId: info.RequestId, UsingGroup: info.UsingGroup, OriginModelName: info.OriginModelName}
		finish := BindUpstreamModelGuard(copyInfo)
		copyInfo.SetUpstreamResponseModelName("wrong")
		finish()
	}
	rows, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	assert.Equal(t, 1, rows[0].ConsecutiveMismatches)
	assert.False(t, rows[0].ChannelDisabled)
}

func TestUpstreamModelGuardMissingModelDoesNotResetStreak(t *testing.T) {
	channel, info, config := setupUpstreamModelGuardServiceTest(t)
	_, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules, FailureThreshold: common.GetPointer(2)}, 42)
	require.NoError(t, err)
	for index, actual := range []string{"wrong", "", "wrong"} {
		info.RequestId = fmt.Sprintf("no-model-%d", index)
		finish := BindUpstreamModelGuard(info)
		info.SetUpstreamResponseModelName(actual)
		finish()
	}
	enabled, err := model.IsChannelEnabledAfterUpstreamModelGuard(t.Context(), channel.Id)
	require.NoError(t, err)
	assert.False(t, enabled)
	rows, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	assert.Equal(t, 2, rows[0].ConsecutiveMismatches)
}

func TestUpstreamModelGuardRetryChannelsShareRequestIdentity(t *testing.T) {
	channel, info, config := setupUpstreamModelGuardServiceTest(t)
	_, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules, FailureThreshold: common.GetPointer(2)}, 42)
	require.NoError(t, err)
	other := model.Channel{Name: "重试渠道", Key: "local", Status: common.ChannelStatusEnabled}
	require.NoError(t, model.DB.Create(&other).Error)
	seed := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelId: other.Id}, UsingGroup: "default", OriginModelName: "client", RequestId: "seed-other"}
	seedFinish := BindUpstreamModelGuard(seed)
	seed.SetUpstreamResponseModelName("wrong")
	seedFinish()
	finish := BindUpstreamModelGuard(info)
	info.SetUpstreamResponseModelName("wrong")
	info.ChannelId = other.Id
	info.SetUpstreamResponseModelName("provider")
	info.ChannelId = channel.Id
	info.SetUpstreamResponseModelName("wrong-again")
	finish()
	rows, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Equal(t, 1, rows[0].ConsecutiveMismatches)
	var streaks []model.UpstreamModelGuardStreak
	require.NoError(t, model.DB.Find(&streaks).Error)
	require.Len(t, streaks, 1)
	assert.Equal(t, channel.Id, streaks[0].ChannelID)
}

func TestUpstreamModelGuardChangedConfigCancelsPendingMatchAndMismatch(t *testing.T) {
	channel, info, config := setupUpstreamModelGuardServiceTest(t)
	config, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules, FailureThreshold: common.GetPointer(2)}, 42)
	require.NoError(t, err)
	finish := BindUpstreamModelGuard(info)
	info.SetUpstreamResponseModelName("wrong")
	finish()
	info.RequestId = "pending-match"
	finish = BindUpstreamModelGuard(info)
	info.SetUpstreamResponseModelName("provider")
	excluded := []int{channel.Id}
	_, err = SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules, ExcludedChannelIDs: &excluded}, 42)
	require.NoError(t, err)
	finish()
	info.SetUpstreamResponseModelName("wrong")
	var streak model.UpstreamModelGuardStreak
	require.NoError(t, model.DB.First(&streak, "channel_id = ?", channel.Id).Error)
	assert.Equal(t, 1, streak.ConsecutiveMismatches)
	_, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
}
