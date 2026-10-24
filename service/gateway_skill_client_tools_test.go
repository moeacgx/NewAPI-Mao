package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewaySkillProtocolsSeparateDeclaredClientTools(t *testing.T) {
	for _, test := range []struct {
		name     string
		format   types.RelayFormat
		request  string
		response string
		history  string
	}{
		{"chat", types.RelayFormatOpenAI,
			`{"model":"test","messages":[{"role":"assistant","content":null,"reasoning_content":"original-reasoning","tool_calls":[{"id":"history","type":"function","function":{"name":"pwsh","arguments":"{}"}}]},{"role":"tool","tool_call_id":"history","content":"history-result"},{"role":"user","content":"review code"}],"tools":[{"type":"function","function":{"name":"pwsh"}}]}`,
			`{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call_client","type":"function","function":{"name":"pwsh","arguments":"{}"}},{"id":"call_read","type":"function","function":{"name":"newapi_skills_read","arguments":"{}"}}]}}]}`,
			"messages"},
		{"responses", types.RelayFormatOpenAIResponses,
			`{"model":"test","input":[{"type":"reasoning","encrypted_content":"original-reasoning"},{"type":"function_call","call_id":"history","name":"pwsh","arguments":"{}"},{"type":"function_call_output","call_id":"history","output":"history-result"},{"role":"user","content":"review code"}],"tools":[{"type":"function","name":"pwsh"}]}`,
			`{"status":"completed","output":[{"type":"function_call","call_id":"call_client","name":"pwsh","arguments":"{}"},{"type":"function_call","call_id":"call_read","name":"newapi_skills_read","arguments":"{}"}]}`,
			"input"},
		{"claude", types.RelayFormatClaude,
			`{"model":"test","max_tokens":2048,"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"thought","signature":"original-reasoning"},{"type":"tool_use","id":"history","name":"pwsh","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"history","content":"history-result"},{"type":"text","text":"review code"}]}],"tools":[{"name":"pwsh","input_schema":{"type":"object"}}]}`,
			`{"role":"assistant","content":[{"type":"tool_use","id":"call_client","name":"pwsh","input":{}},{"type":"tool_use","id":"call_read","name":"newapi_skills_read","input":{}}]}`,
			"messages"},
		{"gemini", types.RelayFormatGemini,
			`{"contents":[{"role":"model","parts":[{"thoughtSignature":"original-reasoning","functionCall":{"id":"history","name":"pwsh","args":{}}}]},{"role":"user","parts":[{"functionResponse":{"id":"history","name":"pwsh","response":{"result":"history-result"}}},{"text":"review code"}]}],"tools":[{"functionDeclarations":[{"name":"pwsh"}]}]}`,
			`{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"id":"call_client","name":"pwsh","args":{}}},{"functionCall":{"id":"call_read","name":"newapi_skills_read","args":{}}}]}}]}`,
			"contents"},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := []byte(test.request)
			clientTools := GatewaySkillClientTools(original, test.format)
			require.Equal(t, map[string]bool{"pwsh": true}, clientTools)
			turn, err := ParseGatewaySkillTurn([]byte(test.response), test.format, clientTools)
			require.NoError(t, err)
			require.Len(t, turn.Calls, 1)
			require.Len(t, turn.ClientCalls, 1)
			assert.Equal(t, GatewaySkillsReadTool, turn.Calls[0].Name)
			assert.Equal(t, "pwsh", turn.ClientCalls[0].Name)
			_, err = ParseGatewaySkillTurn([]byte(test.response), test.format, nil)
			require.ErrorContains(t, err, "未声明工具")
			duplicate := strings.ReplaceAll(test.response, "call_read", "call_client")
			_, err = ParseGatewaySkillTurn([]byte(duplicate), test.format, clientTools)
			require.ErrorContains(t, err, "重复")
			if test.format != types.RelayFormatGemini {
				missing := strings.ReplaceAll(test.response, "call_client", "")
				_, err = ParseGatewaySkillTurn([]byte(missing), test.format, clientTools)
				require.ErrorContains(t, err, "缺失")
			}
			prepared, err := PrepareGatewaySkillRequest(original, test.format, "test", `{"skills":[]}`)
			require.NoError(t, err)
			_, err = ContinueGatewaySkillRequest(prepared, test.format, turn, []GatewaySkillResult{{Call: turn.Calls[0], Content: []byte(`{}`)}})
			require.Error(t, err, "混合私有调用不能回填不配对的工具结果")
			refocused, err := RefocusGatewaySkillRequest(prepared, test.format)
			require.NoError(t, err)
			assert.NotContains(t, string(refocused), "call_client")
			assert.Contains(t, string(refocused), "original-reasoning")
			assert.Contains(t, string(refocused), "history-result")
			before := gjson.GetBytes(prepared, test.history).Array()
			after := gjson.GetBytes(refocused, test.history).Array()
			require.Len(t, after, len(before)+1)
			for i := range before {
				assert.JSONEq(t, before[i].Raw, after[i].Raw)
			}
			assert.Contains(t, after[len(after)-1].Raw, "Private gateway discovery correction")
			assert.JSONEq(t, gjson.GetBytes(prepared, "tools").Raw, gjson.GetBytes(refocused, "tools").Raw)
			assert.Equal(t, test.request, string(original))
		})
	}
	assert.Empty(t, GatewaySkillClientTools([]byte(`{"messages":[{"tool_calls":[{"function":{"name":"pwsh"}}]}]}`), types.RelayFormatOpenAI), "不能从历史调用取得工具声明")
	assert.Equal(t, map[string]bool{"pwsh": true}, GatewaySkillClientTools([]byte(`{"tools":[{"function_declarations":[{"name":"pwsh"}]}]}`), types.RelayFormatGemini))
	refocused, err := RefocusGatewaySkillRequest([]byte(`{"input":"original question"}`), types.RelayFormatOpenAIResponses)
	require.NoError(t, err)
	assert.Equal(t, "original question", gjson.GetBytes(refocused, "input.0.content").String())
}
