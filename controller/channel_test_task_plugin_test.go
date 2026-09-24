package controller

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelProbeTaskPluginReturnsUnsupportedWithoutUpstreamFailure(t *testing.T) {
	group, channel := setupChannelGroupDisplayControllerTestDB(t)
	channel.Type = constant.ChannelTypeTaskPlugin
	channel.Models = "typesafe/jev"
	channel.SetSetting(dto.ChannelSettings{TaskPluginKey: "cloudflare-jev"})
	user := &model.User{Username: "task-plugin-probe", Group: group.Code, Quota: 1000000, Status: common.UserStatusEnabled}
	require.NoError(t, model.DB.Create(user).Error)

	for _, endpoint := range []string{"", string(constant.EndpointTypeOpenAIResponse)} {
		t.Run("端点="+endpoint, func(t *testing.T) {
			result := testChannel(context.Background(), channel, user.Id, "typesafe/jev", endpoint, false)
			require.ErrorContains(t, result.localErr, "channel test is not supported")
			assert.Nil(t, result.newAPIError, "不支持探测不能被记录为上游渠道故障")
			assert.Empty(t, result.upstreamResponseModelName)
			var stored model.User
			require.NoError(t, model.DB.First(&stored, user.Id).Error)
			assert.Equal(t, int64(1000000), stored.Quota, "不支持的探测不消耗用户余额")
		})
	}
}
