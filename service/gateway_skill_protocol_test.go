package service

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewaySkillPrivateRequestsRejectInseparableServerState(t *testing.T) {
	for _, test := range []struct {
		name    string
		format  types.RelayFormat
		request string
	}{
		{"responses-conversation-id", types.RelayFormatOpenAIResponses, `{"model":"test","input":"question","conversation":"conv_original"}`},
		{"responses-conversation-object", types.RelayFormatOpenAIResponses, `{"model":"test","input":"question","conversation":{"id":"conv_original"}}`},
		{"gemini-cached-content", types.RelayFormatGemini, `{"contents":[{"parts":[{"text":"question"}]}],"cachedContent":"cachedContents/original"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := []byte(test.request)
			prepared, err := PrepareGatewaySkillRequest(original, test.format, "test", `{"skills":[]}`)
			require.Error(t, err)
			assert.Nil(t, prepared)
			var apiErr *types.NewAPIError
			require.True(t, errors.As(err, &apiErr))
			assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
			assert.Equal(t, types.ErrorCodeInvalidRequest, apiErr.GetErrorCode())
			assert.Equal(t, test.request, string(original))
		})
	}
	for _, test := range []struct {
		format  types.RelayFormat
		request string
	}{
		{types.RelayFormatOpenAIResponses, `{"model":"test","input":"question","conversation":null,"previous_response_id":"resp_original"}`},
		{types.RelayFormatGemini, `{"contents":[{"parts":[{"text":"question"}]}],"cachedContent":null}`},
		{types.RelayFormatGemini, `{"contents":[{"parts":[{"text":"question"}]}],"cachedContent":""}`},
	} {
		prepared, err := PrepareGatewaySkillRequest([]byte(test.request), test.format, "test", `{"skills":[]}`)
		require.NoError(t, err)
		if test.format == types.RelayFormatOpenAIResponses {
			assert.Equal(t, "resp_original", gjson.GetBytes(prepared, "previous_response_id").String())
		}
	}
}

func TestGatewaySkillProtocolsPreservePrivateContinuation(t *testing.T) {
	for _, test := range []struct {
		name     string
		format   types.RelayFormat
		request  string
		response string
		retained string
		result   string
	}{
		{"chat", types.RelayFormatOpenAI,
			`{"model":"test","messages":[{"role":"user","content":[{"type":"text","text":"question"},{"type":"image_url","image_url":{"url":"https://example.test/image"}}]}],"stream":true,"temperature":0,"unknown":{"value":false},"tools":[{"type":"function","function":{"name":"client_tool"}}],"tool_choice":"required","response_format":{"type":"json_object"}}`,
			`{"choices":[{"message":{"role":"assistant","content":null,"reasoning_content":"private reasoning","tool_calls":[{"id":"call_1","type":"function","function":{"name":"newapi_skills_read","arguments":"{\"template_id\":1,\"skill_id\":\"s1\",\"version\":1}"}}]}}]}`,
			`"reasoning_content":"private reasoning"`, "messages.3.tool_call_id"},
		{"responses", types.RelayFormatOpenAIResponses,
			`{"model":"test","input":"question","stream":true,"temperature":0,"unknown":{"value":false},"include":["reasoning.encrypted_content"],"tools":[{"type":"function","name":"client_tool"}],"tool_choice":"required","text":{"format":{"type":"json_object"}}}`,
			`{"status":"completed","output":[{"type":"reasoning","id":"rs_1","encrypted_content":"encrypted-preserved","summary":[]},{"type":"function_call","id":"fc_1","call_id":"call_1","name":"newapi_skills_read","arguments":"{\"template_id\":1,\"skill_id\":\"s1\",\"version\":1}"}]}`,
			`"encrypted_content":"encrypted-preserved"`, "input.3.call_id"},
		{"claude", types.RelayFormatClaude,
			`{"model":"test","messages":[{"role":"user","content":[{"type":"text","text":"question"}]}],"max_tokens":2048,"stream":true,"temperature":0,"unknown":{"value":false},"tools":[{"name":"client_tool","input_schema":{"type":"object"}}],"tool_choice":{"type":"any"}}`,
			`{"role":"assistant","content":[{"type":"thinking","thinking":"reasoning","signature":"signed-thinking-preserved"},{"type":"tool_use","id":"call_1","name":"newapi_skills_read","input":{"template_id":1,"skill_id":"s1","version":1}}]}`,
			`"signature":"signed-thinking-preserved"`, "messages.2.content.0.tool_use_id"},
		{"gemini", types.RelayFormatGemini,
			`{"contents":[{"role":"user","parts":[{"text":"question"},{"inlineData":{"mimeType":"image/png","data":"YWJj"}}]}],"temperature":0,"unknown":{"value":false},"generationConfig":{"candidateCount":3,"responseMimeType":"application/json","temperature":0},"tools":[{"functionDeclarations":[{"name":"client_tool"}]}]}`,
			`{"candidates":[{"content":{"role":"model","parts":[{"thoughtSignature":"thought-signature-preserved","functionCall":{"id":"call_1","name":"newapi_skills_read","args":{"template_id":1,"skill_id":"s1","version":1}}}]}}]}`,
			`"thoughtSignature":"thought-signature-preserved"`, "contents.2.parts.0.functionResponse.id"},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := []byte(test.request)
			prepared, err := PrepareGatewaySkillRequest(original, test.format, "test", `[{"title":"metadata-only"}]`)
			require.NoError(t, err)
			assert.Equal(t, test.request, string(original))
			assert.Contains(t, string(prepared), "metadata-only")
			assert.NotContains(t, string(prepared), "client_tool")
			assert.NotContains(t, string(prepared), "response_format")
			assert.Equal(t, "0", gjson.GetBytes(prepared, "temperature").Raw)
			assert.Equal(t, "false", gjson.GetBytes(prepared, "unknown.value").Raw)
			if test.format != types.RelayFormatGemini {
				assert.False(t, gjson.GetBytes(prepared, "stream").Bool())
			} else {
				assert.EqualValues(t, 1, gjson.GetBytes(prepared, "generationConfig.candidateCount").Int())
				assert.False(t, gjson.GetBytes(prepared, "generationConfig.responseMimeType").Exists())
			}
			if test.format == types.RelayFormatOpenAIResponses {
				assert.Len(t, gjson.GetBytes(prepared, "include").Array(), 1)
			}
			turn, err := ParseGatewaySkillTurn([]byte(test.response), test.format, nil)
			require.NoError(t, err)
			require.Len(t, turn.Calls, 1)
			continued, err := ContinueGatewaySkillRequest(prepared, test.format, turn,
				[]GatewaySkillResult{{Call: turn.Calls[0], Content: []byte(`{"content":"selected-page","eof":true}`)}})
			require.NoError(t, err)
			assert.Contains(t, string(continued), test.retained)
			assert.Equal(t, "call_1", gjson.GetBytes(continued, test.result).String())
			assert.Contains(t, string(continued), "selected-page")
			assert.Contains(t, string(continued), "question")
			assert.Equal(t, "false", gjson.GetBytes(continued, "unknown.value").Raw)
		})
	}
}

func TestGatewaySkillProtocolsRejectUnmatchedTools(t *testing.T) {
	for _, response := range []string{
		`{`,
		`{"choices":[]}`,
		`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"x","function":{"name":"client_tool","arguments":"{}"}}]}}]}`,
		`{"choices":[{"message":{"role":"assistant","tool_calls":[{"function":{"name":"newapi_skills_list","arguments":"{}"}}]}}]}`,
		`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"x","function":{"name":"newapi_skills_list","arguments":"{}"}},{"id":"x","function":{"name":"newapi_skills_read","arguments":"{}"}}]}}]}`,
	} {
		_, err := ParseGatewaySkillTurn([]byte(response), types.RelayFormatOpenAI, nil)
		require.Error(t, err)
	}
	turn := &GatewaySkillTurn{Calls: []GatewaySkillCall{{ID: "a", Name: GatewaySkillsReadTool}}}
	_, err := ContinueGatewaySkillRequest([]byte(`{"messages":[]}`), types.RelayFormatOpenAI, turn,
		[]GatewaySkillResult{{Call: GatewaySkillCall{ID: "other", Name: GatewaySkillsReadTool}}})
	require.Error(t, err)
}

func TestGatewaySkillPrivateClaudeDoesNotInvokeClientMCP(t *testing.T) {
	original := []byte(`{"model":"test","max_tokens":2048,"messages":[{"role":"user","content":"question"}],"mcp_servers":[{"type":"url","url":"https://example.test/mcp","name":"client-server"}],"tools":[{"type":"mcp_toolset","mcp_server_name":"client-server"}]}`)
	prepared, err := PrepareGatewaySkillRequest(original, types.RelayFormatClaude, "test", `{"skills":[]}`)
	require.NoError(t, err)
	assert.False(t, gjson.GetBytes(prepared, "mcp_servers").Exists())
	assert.NotContains(t, string(prepared), "client-server")
	assert.Len(t, gjson.GetBytes(prepared, "tools").Array(), 3)
	assert.True(t, gjson.GetBytes(original, "mcp_servers").Exists())
}
