package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelledClaudeUsageSettlesWalletOrRefundsReservation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		usage     dto.ClaudeUsage
		wantQuota int
	}{
		{name: "已观测输入", usage: dto.ClaudeUsage{InputTokens: 7}, wantQuota: 7},
		{name: "只有缓存读取", usage: dto.ClaudeUsage{CacheReadInputTokens: 4}, wantQuota: 2},
		{name: "输入输出和缓存读写", usage: dto.ClaudeUsage{InputTokens: 7, OutputTokens: 2, CacheReadInputTokens: 4, CacheCreationInputTokens: 3}, wantQuota: 19},
		{name: "无用量取消退款"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			truncate(t)
			seedUser(t, 1, 1000)
			seedChannel(t, 1)
			ctx, info, _ := newRetryBillingRelayInfo()
			requestCtx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil).WithContext(requestCtx)
			info.IsStream = true
			info.RelayFormat = types.RelayFormatOpenAIResponses
			info.FinalRequestRelayFormat = types.RelayFormatClaude
			info.OriginModelName = "claude-test"
			info.TokenQuotaExempt = true
			info.UserQuota = 1000
			info.UserSetting.QuotaWarningThreshold = 1
			info.StreamStatus = relaycommon.NewStreamStatus()
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, context.Canceled)
			info.PriceData.CompletionRatio = 2
			info.PriceData.CacheRatio = 0.5
			info.PriceData.CacheCreationRatio = 2
			info.PriceData.CacheCreation5mRatio = 2
			info.SetEstimatePromptTokens(1200)
			wallet := &WalletFunding{userId: info.UserId}
			require.NoError(t, wallet.PreConsume(100))
			session := &BillingSession{relayInfo: info, funding: wallet, preConsumedQuota: 100}
			info.Billing = session
			cancel()
			usage := &dto.Usage{UsageSemantic: dto.BillingUsageSemanticAnthropic, BillingUsage: dto.NewClaudeMessagesBillingUsage(&tc.usage)}

			apiErr := PostTextConsumeQuota(ctx, info, usage, nil)

			wantLogType := model.LogTypeConsume
			if tc.wantQuota > 0 {
				require.Nil(t, apiErr)
				assert.False(t, session.NeedsRefund())
				// 结算会话再次关闭或外层清理退款均不能重复扣退余额。
				require.NoError(t, session.Settle(tc.wantQuota))
				session.Refund(ctx)
			} else {
				require.NotNil(t, apiErr)
				assert.Equal(t, types.ErrorCodeDoRequestFailed, apiErr.GetErrorCode())
				assert.True(t, session.NeedsRefund())
				session.Refund(ctx)
				session.Refund(ctx)
				// 生产退款通过异步任务执行；等待实际余额恢复，而非仅断言任务已提交。
				require.EventuallyWithT(t, func(collect *assert.CollectT) {
					var user model.User
					if assert.NoError(collect, model.DB.First(&user, 1).Error) {
						assert.Equal(collect, int64(1000), user.Quota)
					}
				}, 2*time.Second, 10*time.Millisecond)
				wantLogType = model.LogTypeError
			}
			var user model.User
			require.NoError(t, model.DB.First(&user, 1).Error)
			assert.Equal(t, int64(1000-tc.wantQuota), user.Quota)
			var logs []model.Log
			require.NoError(t, model.LOG_DB.Where("user_id = ?", 1).Find(&logs).Error)
			require.Len(t, logs, 1)
			assert.Equal(t, wantLogType, logs[0].Type)
			assert.Equal(t, tc.wantQuota, logs[0].Quota)
			assert.Equal(t, tc.usage.InputTokens, logs[0].PromptTokens)
			assert.Equal(t, tc.usage.OutputTokens, logs[0].CompletionTokens)
			other, err := common.StrToMap(logs[0].Other)
			require.NoError(t, err)
			status, ok := other["stream_status"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "client_gone", status["end_reason"])
		})
	}
}
