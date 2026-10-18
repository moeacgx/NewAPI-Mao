package service

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	channelmetrics "github.com/QuantumNous/new-api/pkg/channel_metrics"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepSeekStandaloneSettlementPreservesWalletAuditAndMetrics(t *testing.T) {
	for _, tc := range []struct {
		name, modelName string
		mode            int
		price, ratio    float64
		quota           int
	}{
		{"搜索自定义价与分组倍率", constant.DeepSeekSearchModel, relayconstant.RelayModeToolsSearch, 0.012, 1.5, 9000},
		{"抓取自定义价与分组倍率", constant.DeepSeekFetchModel, relayconstant.RelayModeToolsFetch, 0.004, 0.5, 1000},
		{"免费搜索仍记录成功请求", constant.DeepSeekSearchModel, relayconstant.RelayModeToolsSearch, 0, 1, 0},
		{"免费抓取仍记录成功请求", constant.DeepSeekFetchModel, relayconstant.RelayModeToolsFetch, 0, 1, 0},
		{"免费分组仍记录成功请求", constant.DeepSeekSearchModel, relayconstant.RelayModeToolsSearch, 0.012, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			truncate(t)
			const id, initialQuota = 98301, 100000
			seedUser(t, id, initialQuota)
			seedToken(t, id, id, "deepseek-settlement", initialQuota)
			seedChannel(t, id)
			collector, _ := installChannelMetricTestRuntime(t)
			ctx := newChannelMetricTestContext(t, tc.name)
			ctx.Request = ctx.Request.Clone(ctx.Request.Context())
			ctx.Request.URL.Path = "/v1/tools/search"
			if tc.mode == relayconstant.RelayModeToolsFetch {
				ctx.Request.URL.Path = "/v1/tools/fetch"
			}
			common.SetContextKey(ctx, constant.ContextKeyOriginalModel, tc.modelName)
			info := newChannelMetricTestRelayInfo()
			info.UserId, info.TokenId, info.TokenKey, info.RequestId = id, id, "deepseek-settlement", tc.name
			info.OriginModelName, info.RelayMode = tc.modelName, tc.mode
			info.ChannelMeta.ChannelId = id
			info.RelayFormat = types.RelayFormatDeepSeekTools
			info.UserSetting.BillingPreference = "wallet_only"
			info.ForcePreConsume = true
			info.PriceData = hosttypes.PriceData{
				UsePrice: true, ModelPrice: tc.price, FreeModel: tc.price == 0,
				GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: tc.ratio},
			}
			require.Nil(t, PreConsumeBilling(ctx, tc.quota, info))
			assert.Equal(t, tc.quota, info.Billing.GetPreConsumedQuota())
			BeginChannelMetricRequest(ctx)
			BindChannelMetricRelayInfo(ctx, info)
			BeginChannelMetricAttempt(ctx, info, id, tc.modelName, 1)

			require.Nil(t, PostTextConsumeQuota(ctx, info, &dto.Usage{}, nil))
			FinishChannelMetricAttempt(ctx, info, nil, false, "")
			ctx.Writer.WriteHeader(http.StatusOK)
			FinishChannelMetricRequest(ctx, info, nil)
			assert.False(t, info.Billing.NeedsRefund())

			var user model.User
			var token model.Token
			var channel model.Channel
			require.NoError(t, model.DB.First(&user, id).Error)
			require.NoError(t, model.DB.First(&token, id).Error)
			require.NoError(t, model.DB.First(&channel, id).Error)
			assert.EqualValues(t, initialQuota-tc.quota, user.Quota)
			assert.EqualValues(t, tc.quota, user.UsedQuota)
			assert.Equal(t, 1, user.RequestCount)
			assert.Equal(t, initialQuota-tc.quota, token.RemainQuota)
			assert.Equal(t, tc.quota, token.UsedQuota)
			assert.EqualValues(t, tc.quota, channel.UsedQuota)
			var logs []model.Log
			require.NoError(t, model.LOG_DB.Where("request_id = ?", tc.name).Find(&logs).Error)
			require.Len(t, logs, 1)
			assert.Equal(t, model.LogTypeConsume, logs[0].Type)
			assert.Equal(t, tc.quota, logs[0].Quota)
			assert.Zero(t, logs[0].PromptTokens)
			assert.Zero(t, logs[0].CompletionTokens)
			var metadata map[string]any
			require.NoError(t, common.UnmarshalJsonStr(logs[0].Other, &metadata))
			assert.Equal(t, tc.price, metadata["model_price"])
			assert.Equal(t, tc.ratio, metadata["group_ratio"])
			assert.Equal(t, BillingSourceWallet, metadata["billing_source"])
			assert.NotContains(t, metadata, "tool_surcharges")

			batch := drainChannelMetricTestBatch(t, collector)
			attempts := channelMetricTestBuckets(batch, channelmetrics.ScopeChannelAttempt, channelmetrics.OutcomeSuccess)
			require.Len(t, attempts, 1)
			assert.EqualValues(t, 1, attempts[0].Counters.EventCount)
			assert.EqualValues(t, tc.quota, attempts[0].Counters.ChargedQuota)
			assert.Zero(t, attempts[0].Counters.InputTokensTotal)
			assert.Zero(t, attempts[0].Counters.OutputTokens)
		})
	}
}
