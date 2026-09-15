package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件仅复制到 9fe0457ee 的 service 包运行，用于复现其与本地计费契约的差异。
// 正常根模块测试不扫描 testdata；不得将失败断言改成认可重复收费或漏计费。
func TestAuditResponsesTerminalOnlyOutputRemainsBillable(t *testing.T) {
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4o"}}
	info.SetEstimatePromptTokens(10)
	accumulator := NewResponsesUsageAccumulator(info)
	var event dto.ResponsesStreamResponse
	require.NoError(t, common.Unmarshal([]byte(`{"type":"response.completed","response":{"id":"resp_terminal","output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}}`), &event))
	accumulator.Observe(&event)
	usage := accumulator.Finish()
	assert.Positive(t, usage.CompletionTokens, "终态已交付实际文本，缺失 usage 时仍应按本地输出回退计费")
	assert.Equal(t, 10, usage.PromptTokens)
}

func TestAuditResponsesDuplicateToolItemsChargeOnce(t *testing.T) {
	info := &relaycommon.RelayInfo{OriginModelName: "gpt-4o"}
	accumulator := NewResponsesUsageAccumulator(info)
	var event dto.ResponsesStreamResponse
	require.NoError(t, common.Unmarshal([]byte(`{"type":"response.output_item.done","output_index":0,"item":{"id":"ws_search_1","type":"web_search_call","status":"completed"}}`), &event))
	accumulator.Observe(&event)
	accumulator.Observe(&event)
	accumulator.Finish()
	require.NotNil(t, info.ResponsesUsageInfo)
	tool := info.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]
	require.NotNil(t, tool)
	assert.Equal(t, 1, tool.CallCount, "同一工具 ID 的重复完成事件只允许收取一次工具费用")
}
