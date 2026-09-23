package billingexpr

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTaskUsageExpressionsPreserveFactTypesAndMissingValues(t *testing.T) {
	for _, tc := range []struct {
		name, expression string
		facts            map[string]any
		want             float64
	}{
		{"数值", `u("input_tokens") * 0.042 / 1000000`, map[string]any{"input_tokens": float64(1000)}, 0.000042},
		{"枚举与布尔", `u("quality") == "high" && u("enabled") ? 0.3 : 0.1`, map[string]any{"quality": "high", "enabled": true}, 0.3},
		{"缺失默认值", `(u("input_tokens") ?? 0) * 0.01`, nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, _, err := RunExprWithRequest(tc.expression, TokenParams{}, RequestInput{Usage: tc.facts})
			require.NoError(t, err)
			assert.InDelta(t, tc.want, result, 1e-12)
		})
	}
	_, _, err := RunExprWithRequest(`u("missing") * 0.01`, TokenParams{}, RequestInput{})
	require.Error(t, err)
}

func TestTaskUsageSettlementKeepsChatQuotaUnits(t *testing.T) {
	for _, tc := range []struct {
		name, expression string
		task             bool
		params           TokenParams
		usage            map[string]any
	}{
		{"任务美元", `u("input_tokens") * 0.042 / 1000000`, true, TokenParams{}, map[string]any{"input_tokens": float64(1000)}},
		{"聊天每百万token", `p * 0.042`, false, TokenParams{P: 1000}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := &BillingSnapshot{ExprString: tc.expression, ExprHash: ExprHashString(tc.expression), ExprVersion: 1, TaskUsageBilling: tc.task, QuotaPerUnit: 500000, GroupRatio: 2}
			result, err := ComputeTieredQuotaWithRequest(snapshot, tc.params, RequestInput{Usage: tc.usage})
			require.NoError(t, err)
			assert.InDelta(t, 21, result.ActualQuotaBeforeGroup, 1e-9)
			assert.Equal(t, 42, result.ActualQuotaAfterGroup)
			assert.Nil(t, result.Clamp)
		})
	}
}
