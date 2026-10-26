package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpstreamModelGuardIgnoresNonBodyEvidenceWithoutChangingStreak(t *testing.T) {
	channel, info, config := setupUpstreamModelGuardServiceTest(t)
	_, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{
		ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules, FailureThreshold: common.GetPointer(3),
	}, 42)
	require.NoError(t, err)
	finish := BindUpstreamModelGuard(info)
	info.SetUpstreamResponseModelName("wrong-provider")
	finish()

	for _, source := range []string{constant.UpstreamModelSourceCodexFasterModel, "unknown"} {
		next := &relaycommon.RelayInfo{ChannelMeta: info.ChannelMeta, UsingGroup: info.UsingGroup, OriginModelName: info.OriginModelName, RequestId: source}
		finish := BindUpstreamModelGuard(next)
		next.OnUpstreamModelEvidence(next, relaycommon.UpstreamModelEvidence{Model: "wrong-provider", Source: source})
		next.OnUpstreamModelEvidence(next, relaycommon.UpstreamModelEvidence{Model: "provider", Source: source})
		finish()
	}
	var streak model.UpstreamModelGuardStreak
	require.NoError(t, model.DB.Where("channel_id = ?", channel.Id).First(&streak).Error)
	assert.Equal(t, 1, streak.ConsecutiveMismatches)
	_, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	enabled, err := model.IsChannelEnabledAfterUpstreamModelGuard(t.Context(), channel.Id)
	require.NoError(t, err)
	assert.True(t, enabled)
}
