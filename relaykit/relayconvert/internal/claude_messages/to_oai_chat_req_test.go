package claudemessages

import (
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeToChatToolChoice(t *testing.T) {
	for _, tt := range []struct {
		name, raw string
		choice    any
		parallel  *bool
		wantError bool
	}{
		{name: "未指定", raw: `{}`},
		{name: "空值", raw: `{"tool_choice":null}`},
		{name: "自动", raw: `{"tool_choice":{"type":"auto"}}`, choice: "auto"},
		{name: "显式允许并行", raw: `{"tool_choice":{"type":"auto","disable_parallel_tool_use":false}}`, choice: "auto", parallel: kitutil.GetPointer(true)},
		{name: "必须调用且禁止并行", raw: `{"tool_choice":{"type":"any","disable_parallel_tool_use":true}}`, choice: "required", parallel: kitutil.GetPointer(false)},
		{name: "指定工具", raw: `{"tool_choice":{"type":"tool","name":"lookup"}}`, choice: map[string]any{"type": "function", "function": map[string]any{"name": "lookup"}}},
		{name: "不调用", raw: `{"tool_choice":{"type":"none"}}`, choice: "none"},
		{name: "无效类型", raw: `{"tool_choice":{"type":"other"}}`, wantError: true},
		{name: "缺少名称", raw: `{"tool_choice":{"type":"tool"}}`, wantError: true},
		{name: "非对象", raw: `{"tool_choice":"auto"}`, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var req dto.ClaudeRequest
			require.NoError(t, kitutil.Unmarshal([]byte(tt.raw), &req))
			out, err := ClaudeMessagesRequestToOpenAIChat(req, nil)
			if tt.wantError {
				require.ErrorContains(t, err, "tool_choice")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.choice, out.ToolChoice)
			assert.Equal(t, tt.parallel, out.ParallelTooCalls)
			encoded, err := kitutil.Marshal(out)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, kitutil.Unmarshal(encoded, &payload))
			if tt.parallel == nil {
				assert.NotContains(t, payload, "parallel_tool_calls")
			} else {
				assert.Equal(t, *tt.parallel, payload["parallel_tool_calls"])
			}
		})
	}
}
