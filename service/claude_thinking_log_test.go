package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeThinkingUsageLog(t *testing.T) {
	originalPassThrough := model_setting.GetGlobalSettings().PassThroughRequestEnabled
	t.Cleanup(func() { model_setting.GetGlobalSettings().PassThroughRequestEnabled = originalPassThrough })
	tests := []struct {
		name, body, expected string
	}{
		{"明确等级优先", `{"output_config":{"effort":"max"},"thinking":{"type":"adaptive"}}`, "max"},
		{"思考预算", `{"thinking":{"type":"enabled","budget_tokens":16000}}`, "thinking:16000"},
		{"自适应", `{"thinking":{"type":"adaptive"}}`, "adaptive"},
		{"启用无预算", `{"thinking":{"type":"enabled"}}`, "thinking"},
		{"显式关闭", `{"thinking":{"type":"disabled","budget_tokens":16000}}`, "disabled"},
		{"无参数", `{}`, ""},
	}
	for _, tt := range tests {
		for _, mode := range []string{"普通", "渠道透传", "全局透传"} {
			t.Run(tt.name+"/"+mode, func(t *testing.T) {
				model_setting.GetGlobalSettings().PassThroughRequestEnabled = mode == "全局透传"
				ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
				ctx.Request = httptest.NewRequest("POST", "/v1/messages", strings.NewReader(tt.body))
				common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeAnthropic)
				common.SetContextKey(ctx, constant.ContextKeyChannelSetting, dto.ChannelSettings{PassThroughBodyEnabled: mode == "渠道透传"})
				var request dto.ClaudeRequest
				require.NoError(t, common.Unmarshal([]byte(tt.body), &request))
				info, err := relaycommon.GenRelayInfo(ctx, types.RelayFormatClaude, &request, nil)
				require.NoError(t, err)
				info.InitChannelMeta(ctx)
				other := GenerateTextOtherInfo(ctx, info, 1, 1, 1, 0, 0, 0, 1)
				if tt.expected == "" {
					assert.NotContains(t, other, "reasoning_effort")
				} else {
					assert.Equal(t, tt.expected, other["reasoning_effort"])
				}
				if request.Thinking != nil {
					assert.Equal(t, request.Thinking.Type, other["thinking_type"])
					if request.Thinking.BudgetTokens != nil {
						assert.Equal(t, *request.Thinking.BudgetTokens, other["thinking_budget_tokens"])
					} else {
						assert.NotContains(t, other, "thinking_budget_tokens")
					}
				}
				assert.NotContains(t, info.ReasoningEffort, "thinking", "展示值不能污染协议转换")
			})
		}
	}
}

func TestThinkingLogDoesNotChangeMaxRuleSettlement(t *testing.T) {
	expr := `tier("base", p) * (param("output_config.effort") == "max" ? 3 : 1)`
	for _, tt := range []struct {
		name, body string
		quota      int
		matched    bool
	}{
		{"明确max", `{"output_config":{"effort":"max"},"thinking":{"type":"adaptive"}}`, 150, true},
		{"预算不是max", `{"thinking":{"type":"enabled","budget_tokens":16000}}`, 50, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				RelayFormat: types.RelayFormatClaude,
				TieredBillingSnapshot: &billingexpr.BillingSnapshot{
					BillingMode: "tiered_expr", ExprString: expr, ExprHash: billingexpr.ExprHashString(expr), GroupRatio: 1, QuotaPerUnit: 500000,
				},
				BillingRequestInput: &billingexpr.RequestInput{Body: []byte(tt.body)},
			}
			info.CaptureThinkingLogBody([]byte(tt.body))
			// 模拟出站覆盖删除思考参数：日志随出站变化，计费仍读冻结的入站请求。
			info.CaptureThinkingLogBody([]byte(`{}`))
			ok, quota, result := TryTieredSettle(info, billingexpr.TokenParams{P: 100})
			require.True(t, ok)
			require.NotNil(t, result)
			assert.Equal(t, tt.quota, quota)
			require.Len(t, result.RequestRules, 1)
			assert.Equal(t, tt.matched, result.RequestRules[0].Matched)
			assert.Equal(t, 3.0, result.RequestRules[0].Multiplier)
			assert.Equal(t, tt.body, string(info.BillingRequestInput.Body))
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			info.ChannelMeta = &relaycommon.ChannelMeta{}
			info.ReasoningEffort = "max"
			assert.NotContains(t, GenerateTextOtherInfo(ctx, info, 1, 1, 1, 0, 0, 0, 1), "reasoning_effort", "空快照不得回退到过时等级")
		})
	}
}
