package common

import (
	"net/http/httptest"
	"testing"

	common2 "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThinkingLogCapturesFinalProtocolAndDeletion(t *testing.T) {
	tests := []struct {
		name           string
		format         types.RelayFormat
		body, expected string
	}{
		{"覆盖预算", types.RelayFormatClaude, `{"thinking":{"type":"enabled","budget_tokens":8192}}`, "thinking:8192"},
		{"明确等级优先", types.RelayFormatClaude, `{"thinking":{"type":"adaptive"},"output_config":{"effort":"max"}}`, "max"},
		{"adaptive不猜预算等级", types.RelayFormatClaude, `{"thinking":{"type":"adaptive","budget_tokens":16000}}`, "adaptive"},
		{"关闭不受残留预算影响", types.RelayFormatClaude, `{"thinking":{"type":"disabled","budget_tokens":16000}}`, "disabled"},
		{"删除参数", types.RelayFormatClaude, `{}`, ""},
		{"OpenRouter转换后预算", types.RelayFormatOpenAI, `{"reasoning":{"enabled":true,"max_tokens":16000}}`, "thinking:16000"},
		{"OpenRouter明确关闭", types.RelayFormatOpenAI, `{"reasoning":{"enabled":false,"max_tokens":16000}}`, "disabled"},
		{"Responses转换等级", types.RelayFormatOpenAIResponses, `{"reasoning":{"effort":"high"}}`, "high"},
		{"忽略非整数预算", types.RelayFormatClaude, `{"thinking":{"type":"enabled","budget_tokens":1.5}}`, "thinking"},
		{"忽略非字符串等级", types.RelayFormatClaude, `{"output_config":{"effort":123}}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &RelayInfo{RelayFormat: tt.format, ReasoningEffort: "old-effort", ThinkingLog: &ThinkingLogSnapshot{Effort: "old-effort"}}
			body := []byte(tt.body)
			info.CaptureThinkingLogBody(body)
			require.NotNil(t, info.ThinkingLog)
			assert.Equal(t, tt.expected, info.ThinkingLog.DisplayEffort())
			assert.Equal(t, "old-effort", info.ReasoningEffort, "日志采集不得写协议转换元数据")
			assert.Equal(t, tt.body, string(body))
		})
	}
}

func TestThinkingLogDoesNotReplaceUnrelatedProviderEffort(t *testing.T) {
	info := &RelayInfo{RelayFormat: types.RelayFormatOpenAI, ReasoningEffort: "high"}
	info.CaptureThinkingLogBody([]byte(`{"model":"provider-model"}`))
	assert.Nil(t, info.ThinkingLog)
	assert.Equal(t, "high", info.ReasoningEffort)
}

func TestThinkingLogRetryRestoresCurrentRequest(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/messages", nil)
	budget := 16000
	request := &dto.ClaudeRequest{Thinking: &dto.Thinking{Type: "enabled", BudgetTokens: &budget}}
	info := &RelayInfo{RelayFormat: types.RelayFormatClaude, Request: request}
	info.InitChannelMeta(ctx)
	require.NotNil(t, info.ThinkingLog)
	info.CaptureThinkingLogBody([]byte(`{"output_config":{"effort":"max"}}`))
	assert.Equal(t, "max", info.ThinkingLog.DisplayEffort())
	info.InitChannelMeta(ctx)
	assert.Equal(t, "thinking:16000", info.ThinkingLog.DisplayEffort())
	*request.Thinking.BudgetTokens = 8000
	assert.Equal(t, "thinking:16000", info.ThinkingLog.DisplayEffort(), "快照不得引用可变请求参数")
	info.Request = &dto.ClaudeRequest{}
	info.InitChannelMeta(ctx)
	assert.Empty(t, info.ThinkingLog.DisplayEffort())
	info.Request = &dto.GeneralOpenAIRequest{}
	info.InitChannelMeta(ctx)
	assert.Nil(t, info.ThinkingLog)
}

func TestThinkingLogOpenAIRequestPreservesEffortAndBudget(t *testing.T) {
	var request dto.GeneralOpenAIRequest
	require.NoError(t, common2.Unmarshal([]byte(`{"model":"anthropic/claude","reasoning_effort":"max","thinking":{"type":"enabled","budget_tokens":16000}}`), &request))
	info := &RelayInfo{Request: &request}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	info.InitChannelMeta(ctx)
	require.NotNil(t, info.ThinkingLog)
	assert.Equal(t, "max", info.ThinkingLog.DisplayEffort())
	require.NotNil(t, info.ThinkingLog.BudgetTokens)
	assert.Equal(t, 16000, *info.ThinkingLog.BudgetTokens)
}
