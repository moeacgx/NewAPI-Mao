package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewaySkillChatStreamPreservesFragmentedToolsAndReasoning(t *testing.T) {
	response := ": heartbeat\r\n\r\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"reasoning_content\":\"private \",\"tool_calls\":[{\"index\":0,\"id\":\"call_\",\"type\":\"function\",\"function\":{\"name\":\"newapi_skills_\",\"arguments\":\"{\"}}]}}]}\r\n\r\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"reasoning\",\"tool_calls\":[{\"index\":1,\"id\":\"client\",\"type\":\"function\",\"function\":{\"name\":\"client_lookup\",\"arguments\":\"{}\"}},{\"index\":0,\"id\":\"list\",\"function\":{\"name\":\"list\",\"arguments\":\"}\"}}]}}]}\r\n\r\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\r\n\r\n" +
		"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5}}\r\n\r\n" +
		"data: [DONE]\r\n\r\n"
	turn, err := ParseGatewaySkillChatStream([]byte(response), map[string]bool{"client_lookup": true})
	require.NoError(t, err)
	require.Len(t, turn.Calls, 1)
	assert.Equal(t, "call_list", turn.Calls[0].ID)
	assert.Equal(t, GatewaySkillsListTool, turn.Calls[0].Name)
	assert.JSONEq(t, "{}", string(turn.Calls[0].Arguments))
	require.Len(t, turn.ClientCalls, 1)
	assert.Equal(t, "client", turn.ClientCalls[0].ID)
	assert.Equal(t, "private reasoning", gjson.GetBytes(turn.Assistant, "reasoning_content").String())
}

func TestGatewaySkillChatStreamPreservesCompleteReadyMessage(t *testing.T) {
	response := "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"reasoning\":\"think\",\"content\":\"SKILL_\"}}],\n" +
		"data: \"object\":\"chat.completion.chunk\"}\n\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"CONTEXT_READY\"},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	turn, err := ParseGatewaySkillChatStream([]byte(response), nil)
	require.NoError(t, err)
	assert.Empty(t, turn.Calls)
	assert.Equal(t, "SKILL_CONTEXT_READY", gjson.GetBytes(turn.Assistant, "content").String())
	assert.Equal(t, "think", gjson.GetBytes(turn.Assistant, "reasoning").String())
}

func TestGatewaySkillChatStreamRejectsIncompleteOrInvalidOutput(t *testing.T) {
	start := "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"ready\"}}]}\n\n"
	finish := "data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"
	done := "data: [DONE]\n\n"
	for _, test := range []struct{ name, response string }{
		{"malformed-json", "data: {broken}\n\n" + done},
		{"missing-finish", start + done},
		{"missing-done", start + finish},
		{"truncated-frame", start + finish + strings.TrimSuffix(done, "\n")},
		{"length", start + strings.Replace(finish, "stop", "length", 1) + done},
		{"content-filter", start + strings.Replace(finish, "stop", "content_filter", 1) + done},
		{"multiple-choices", start + strings.Replace(finish, "\"index\":0", "\"index\":1", 1) + done},
		{"invalid-role", strings.Replace(start, "assistant", "user", 1) + finish + done},
		{"invalid-delta", strings.Replace(start, "\"content\":\"ready\"", "\"content\":123", 1) + finish + done},
		{"data-after-finish", start + finish + start + done},
		{"data-after-done", start + finish + done + start},
		{"error-event", "event: error\ndata: {\"error\":{\"message\":\"failed\"}}\n\n" + done},
		{"no-choice", "data: {\"choices\":[]}\n\n" + done},
	} {
		t.Run(test.name, func(t *testing.T) {
			turn, err := ParseGatewaySkillChatStream([]byte(test.response), nil)
			require.Error(t, err)
			assert.Nil(t, turn)
		})
	}
}

func TestGatewaySkillChatStreamRejectsInvalidTools(t *testing.T) {
	tool := `{"index":0,"id":"call","type":"function","function":{"name":"newapi_skills_list","arguments":"{}"}}`
	for _, test := range []struct{ name, tools string }{
		{"duplicate-id", tool + "," + strings.Replace(tool, `"index":0`, `"index":1`, 1)},
		{"invalid-arguments", strings.Replace(tool, `"arguments":"{}"`, `"arguments":"{"`, 1)},
		{"unknown-tool", strings.Replace(tool, "newapi_skills_list", "unknown_tool", 1)},
		{"missing-id", strings.Replace(tool, `"id":"call",`, "", 1)},
		{"missing-index", strings.Replace(tool, `"index":0,`, "", 1)},
		{"negative-index", strings.Replace(tool, `"index":0`, `"index":-1`, 1)},
		{"too-many-tools", strings.Join([]string{tool, strings.Replace(tool, `"index":0`, `"index":8`, 1)}, ",")},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := fmt.Sprintf("data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"tool_calls\":[%s]},\"finish_reason\":\"tool_calls\"}]}\n\ndata: [DONE]\n\n", test.tools)
			turn, err := ParseGatewaySkillChatStream([]byte(response), nil)
			require.Error(t, err)
			assert.Nil(t, turn)
		})
	}
}
