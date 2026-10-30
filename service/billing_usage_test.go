package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/types"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponsesCacheBillingMatchesClientUsage(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "JSON", true: "SSE"}[stream], func(t *testing.T) {
			var response dto.OpenAIResponsesResponse
			require.NoError(t, common.Unmarshal([]byte(`{"id":"resp_cache","model":"gpt-test","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}],"usage":{"input_tokens":100,"output_tokens":20,"total_tokens":120,"input_tokens_details":{"cached_tokens":70}}}`), &response))
			var usage *dto.Usage
			var clientUsage *dto.Usage
			if stream {
				state := relayconvert.NewResponsesToChatStreamState("gpt-test", true)
				for _, raw := range []string{
					`{"type":"response.created","response":{"id":"resp_cache","model":"gpt-test","usage":null}}`,
					`{"type":"response.output_text.delta","delta":"hello"}`,
				} {
					var event dto.ResponsesStreamResponse
					require.NoError(t, common.UnmarshalJsonStr(raw, &event))
					_, err := relayconvert.ResponsesStreamEventToChatChunks(&event, state)
					require.NoError(t, err)
				}
				chunks, err := relayconvert.ResponsesStreamEventToChatChunks(&dto.ResponsesStreamResponse{Type: "response.completed", Response: &response}, state)
				require.NoError(t, err)
				usage = state.Usage
				for _, chunk := range chunks {
					if chunk.Usage != nil {
						clientUsage = chunk.Usage
					}
				}
			} else {
				chat, convertedUsage, err := relayconvert.ResponsesResponseToChatCompletionsResponse(&response, "chat_cache")
				require.NoError(t, err)
				usage = convertedUsage
				clientUsage = &chat.Usage
			}
			require.NotNil(t, clientUsage)
			assert.Equal(t, 70, clientUsage.PromptTokensDetails.CachedTokens)
			require.NotNil(t, usage.BillingUsage)
			require.NotNil(t, usage.BillingUsage.OpenAIUsage)
			assert.Zero(t, usage.BillingUsage.OpenAIUsage.PromptTokensDetails.CachedTokens)
			assert.Equal(t, 70, usage.BillingUsage.OpenAIUsage.InputTokensDetails.CachedTokens)

			// 与 PostTextConsumeQuota 一致，先选实际计费快照，再共享给摘要、表达式和日志。
			effective := effectiveBillingUsage(usage)
			assert.Equal(t, 70, effective.PromptTokensDetails.CachedTokens)
			assert.True(t, effective.PromptTokensDetails.HasCachedTokens)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-test"},
				RelayFormat: types.RelayFormatOpenAI, FinalRequestRelayFormat: types.RelayFormatOpenAIResponses,
				OriginModelName: "gpt-test", StartTime: time.Now(), IsStream: stream,
				PriceData: hosttypes.PriceData{
					ModelRatio: 1, CompletionRatio: 2, CacheRatio: 0.1,
					GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1},
				},
			}
			summary := calculateTextQuotaSummary(ctx, info, effective)
			assert.Equal(t, 100, summary.PromptTokens)
			assert.Equal(t, 20, summary.CompletionTokens)
			assert.Equal(t, 120, summary.TotalTokens)
			assert.Equal(t, 70, summary.CacheTokens)
			assert.Equal(t, 77, summary.Quota, "(100-70)+70*0.1+20*2")
			other := GenerateTextOtherInfo(ctx, info, summary.ModelRatio, summary.GroupRatio, summary.CompletionRatio, summary.CacheTokens, summary.CacheRatio, summary.ModelPrice, 0)
			appendUsageBillingPathForLog(other, false, usage)
			assert.Equal(t, 70, other["cache_tokens"])
			assert.Equal(t, 0.1, other["cache_ratio"])
			assert.Equal(t, usageBillingPathOpenAI, other["admin_info"].(map[string]interface{})["usage_billing_path"])

			const expression = "p*2+c*4+cr*0.2"
			info.TieredBillingSnapshot = &billingexpr.BillingSnapshot{
				BillingMode: "tiered_expr", ExprString: expression, ExprHash: billingexpr.ExprHashString(expression),
				QuotaPerUnit: 500000, GroupRatio: 1,
			}
			params := BuildTieredTokenParams(effective, summary.IsClaudeUsageSemantic, billingexpr.UsedVars(expression))
			assert.Equal(t, float64(70), params.CR)
			assert.Equal(t, float64(30), params.P)
			assert.Equal(t, float64(100), params.Len)
			ok, quota, result := TryTieredSettle(info, params)
			require.True(t, ok)
			require.NotNil(t, result, "表达式应真实求值，不能走估算回退")
			assert.Equal(t, summary.Quota, quota)
		})
	}
}

func TestEffectiveBillingUsageCachePresenceAndSnapshotIsolation(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		present bool
	}{
		{"input cache", `{"input_tokens_details":{"cached_tokens":70}}`, 70, true},
		{"missing details", `{}`, 0, false},
		{"missing cached field", `{"input_tokens_details":{"image_tokens":3}}`, 0, false},
		{"input explicit zero", `{"input_tokens_details":{"cached_tokens":0}}`, 0, true},
		{"prompt wins", `{"prompt_tokens_details":{"cached_tokens":12},"input_tokens_details":{"cached_tokens":70}}`, 12, true},
		{"prompt explicit zero wins", `{"prompt_tokens_details":{"cached_tokens":0},"input_tokens_details":{"cached_tokens":70}}`, 0, true},
		{"missing prompt cache", `{"prompt_tokens_details":{"image_tokens":3},"input_tokens_details":{"cached_tokens":70}}`, 70, true},
		{"legacy nonzero prompt", `{"input_tokens_details":{"cached_tokens":0}}`, 12, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var source dto.Usage
			require.NoError(t, common.UnmarshalJsonStr(tt.raw, &source))
			source.InputTokens, source.OutputTokens = 100, 20
			source.PromptTokensDetails.AudioTokens = 2
			source.OutputTokensDetails = &dto.OutputTokenDetails{ReasoningTokens: 5}
			if tt.name == "legacy nonzero prompt" {
				source.PromptTokensDetails.CachedTokens = 12
			}
			snapshot := dto.NewOpenAIResponsesBillingUsage(&source)
			before := dto.CloneBillingUsage(snapshot)
			// 客户端展示数据不作为原生计费快照的缓存补丁。
			client := &dto.Usage{PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 99}, BillingUsage: snapshot}
			usage := effectiveBillingUsage(client)
			assert.Equal(t, tt.want, usage.PromptTokensDetails.CachedTokens)
			assert.Equal(t, tt.present, usage.PromptTokensDetails.HasCachedTokens)
			assert.Equal(t, 2, usage.PromptTokensDetails.AudioTokens)
			assert.Equal(t, source.PromptTokensDetails.ImageTokens, usage.PromptTokensDetails.ImageTokens)
			assert.Equal(t, 100, usage.PromptTokens)
			assert.Equal(t, 20, usage.CompletionTokens)
			assert.Equal(t, before, snapshot)
			if usage.InputTokensDetails != nil {
				usage.InputTokensDetails.CachedTokens = 321
				assert.Equal(t, before, snapshot)
				assert.Equal(t, before, usage.BillingUsage)
				snapshot.OpenAIUsage.InputTokensDetails.CachedTokens = 123
				assert.Equal(t, 321, usage.InputTokensDetails.CachedTokens)
				assert.Equal(t, before, usage.BillingUsage)
			}
			usage.OutputTokensDetails.ReasoningTokens = 33
			assert.Equal(t, 5, snapshot.OpenAIUsage.OutputTokensDetails.ReasoningTokens)
			assert.Equal(t, 5, usage.BillingUsage.OpenAIUsage.OutputTokensDetails.ReasoningTokens)
		})
	}
	t.Run("no snapshot", func(t *testing.T) {
		usage := &dto.Usage{PromptTokens: 100, PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 70}}
		assert.Same(t, usage, effectiveBillingUsage(usage))
		assert.Nil(t, effectiveBillingUsage(nil))
	})
}
