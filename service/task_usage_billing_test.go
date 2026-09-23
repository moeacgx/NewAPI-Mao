package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

func TestEvaluateTaskCompletionUsageUsesFrozenContract(t *testing.T) {
	expression := `tier(u("input_tokens") > 2000 ? "large" : "small", u("input_tokens") * 0.042 / 1000000) * u("factor")`
	snapshot := &billingexpr.BillingSnapshot{ExprString: expression, ExprHash: billingexpr.ExprHashString(expression), ExprVersion: 1, TaskUsageBilling: true, QuotaPerUnit: 500000, GroupRatio: 2, EstimatedTier: "large", UsageFacts: map[string]any{"input_tokens": float64(65536), "factor": float64(1)}}
	result, facts, err := EvaluateTaskCompletionUsage(snapshot, map[string]any{"input_tokens": float64(1000)})
	require.NoError(t, err)
	assert.Equal(t, 42, result.ActualQuotaAfterGroup)
	assert.Equal(t, "small", result.MatchedTier)
	assert.True(t, result.CrossedTier)
	assert.Equal(t, map[string]any{"input_tokens": float64(1000), "factor": float64(1)}, facts)
	assert.Equal(t, float64(65536), snapshot.UsageFacts["input_tokens"])
	assert.Equal(t, "large", snapshot.EstimatedTier)
	facts["factor"] = float64(2)
	assert.Equal(t, float64(1), snapshot.UsageFacts["factor"])
}

func TestEvaluateTaskCompletionUsageRejectsInvalidCostAndAuditsSaturation(t *testing.T) {
	expression := `u("cost")`
	snapshot := &billingexpr.BillingSnapshot{ExprString: expression, ExprHash: billingexpr.ExprHashString(expression), TaskUsageBilling: true, QuotaPerUnit: 500000, GroupRatio: 1}
	for _, tc := range []struct {
		name                 string
		cost                 float64
		want                 int
		wantError, wantClamp bool
	}{
		{name: "零用量全额差额退款", cost: 0, want: 0},
		{name: "负费用", cost: -1, wantError: true},
		{name: "非数值费用", cost: math.NaN(), wantError: true},
		{name: "结算超限饱和并审计", cost: 1e20, want: common.MaxQuota, wantClamp: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, _, err := EvaluateTaskCompletionUsage(snapshot, map[string]any{"cost": tc.cost})
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, result.ActualQuotaAfterGroup)
			if tc.wantClamp {
				require.NotNil(t, result.Clamp)
				assert.Equal(t, common.QuotaClampOverflow, result.Clamp.Kind)
			} else {
				assert.Nil(t, result.Clamp)
			}
		})
	}
	_, _, err := EvaluateTaskCompletionUsage(nil, nil)
	require.Error(t, err)
}
