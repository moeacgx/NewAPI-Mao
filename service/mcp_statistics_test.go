package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPStatisticsPreserveAnalyticsMetricsAndCoverage(t *testing.T) {
	rate := 0.75
	input := dto.ChannelAnalyticsSummaryResponse{
		Summary: dto.ChannelAnalyticsSummary{FinalRequestCount: 8, ChannelAttemptCount: 12, UpstreamCallCount: 11, FailedAttemptCount: 3, ClientSuccessRate: &rate, CacheReadTokens: 64, ChargedQuota: 25},
		Meta:    dto.ChannelAnalyticsMeta{GeneratedAt: 123, Partial: true, DetailAvailable: false, RuntimeFlushFailureCount: 2},
	}
	raw, err := common.Marshal(map[string]any{"success": true, "data": input})
	require.NoError(t, err)
	projected, ok, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_analytics_summary", Category: "channels"}, raw, 200)
	require.NoError(t, err)
	require.True(t, ok)
	var output struct {
		Data dto.ChannelAnalyticsSummaryResponse `json:"data"`
	}
	require.NoError(t, common.Unmarshal(projected, &output))
	assert.Equal(t, input, output.Data, "统计分子、分母、费用和覆盖范围不能在脱敏时丢失")
}

func TestMCPStatisticsPreserveRevenueAmounts(t *testing.T) {
	input := model.RevenueStatsResponse{
		Summary: model.RevenueSummary{
			Cash:                 []model.RevenueCashTotal{{Currency: "USD", Amount: 18.5, Count: 1}, {Currency: "CNY", Amount: 10, Count: 1}},
			UnresolvedCashCount:  0,
			TotalRedemptionQuota: 200,
			TotalRedemptionCount: 1,
		},
		DataPoints: []model.RevenueDataPoint{{
			Timestamp:           123,
			Cash:                []model.RevenueCashTotal{{Currency: "USD", Amount: 18.5, Count: 1}, {Currency: "CNY", Amount: 10, Count: 1}},
			UnresolvedCashCount: 0,
			RedemptionQuota:     200,
			RedemptionCount:     1,
		}},
	}
	raw, err := common.Marshal(map[string]any{"success": true, "data": input})
	require.NoError(t, err)
	projected, ok, err := ProjectMCPResponse(MCPTool{Name: "newapi_revenue", Category: "usage"}, raw, 200)
	require.NoError(t, err)
	require.True(t, ok)
	var output struct {
		Data model.RevenueStatsResponse `json:"data"`
	}
	require.NoError(t, common.Unmarshal(projected, &output))
	assert.Equal(t, input, output.Data)
}

func TestMCPStatisticsDoNotExposeRawFailuresAndNotificationTargets(t *testing.T) {
	for _, tool := range []string{"newapi_channel_failures", "newapi_notification_deliveries"} {
		raw := []byte(`{"success":true,"data":{"items":[{"event_id":1,"event_type":"ticket_user_message","attempt_count":2,"status_code":429,"last_error":"private-error","error_summary":"private-error","payload":{"content":"private-message"},"chat_id":"private-target","template":"private-template","bot_token":"private-key"}]}}`)
		projected, ok, err := ProjectMCPResponse(MCPTool{Name: tool, Category: "notifications"}, raw, 200)
		require.NoError(t, err)
		require.True(t, ok)
		assert.NotContains(t, string(projected), "private-")
		assert.Contains(t, string(projected), `"event_id":1`)
	}
}
