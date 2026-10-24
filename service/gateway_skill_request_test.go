package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type gatewaySkillRequestFixture struct {
	name       string
	format     types.RelayFormat
	historyKey string
	history    string
	system     string
	response   string
	addedItems int
}

func gatewaySkillRequestFixtures() []gatewaySkillRequestFixture {
	return []gatewaySkillRequestFixture{
		{
			name: "chat", format: types.RelayFormatOpenAI, historyKey: "messages", addedItems: 2,
			history:  `{"role":"assistant","content":null,"reasoning_content":"original-reasoning","tool_calls":[{"id":"old","type":"function","function":{"name":"pwsh","arguments":"{}"}}],"vendor_counter":9007199254740993}, {"role":"tool","tool_call_id":"old","content":"original-result"}, {"role":"user","content":[{"type":"text","text":"task"},{"type":"image_url","image_url":{"url":"data:image/png;base64,YWJj"}},{"type":"input_audio","input_audio":{"data":"YWJj","format":"wav"}},{"type":"file","file":{"file_data":"data:application/pdf;base64,YWJj"}}],"vendor_zero":-0}`,
			response: `{"choices":[{"message":{"role":"assistant","reasoning_content":"private-reasoning","content":null,"vendor":{"zero":0,"flag":false},"tool_calls":[{"id":"call_page","type":"function","function":{"name":"newapi_skills_read","arguments":"{\"offset\":24576}"}}]}}]}`,
		},
		{
			name: "responses", format: types.RelayFormatOpenAIResponses, historyKey: "input", addedItems: 3,
			history:  `{"type":"reasoning","encrypted_content":"original-reasoning","vendor_counter":9007199254740993}, {"type":"function_call","call_id":"old","name":"pwsh","arguments":"{}"}, {"type":"function_call_output","call_id":"old","output":"original-result"}, {"role":"user","content":[{"type":"input_text","text":"task"},{"type":"input_image","image_url":"data:image/png;base64,YWJj"},{"type":"input_file","file_data":"data:application/pdf;base64,YWJj"}],"vendor_zero":-0}`,
			response: `{"status":"completed","output":[{"type":"reasoning","id":"private-reasoning","encrypted_content":"private-encrypted","summary":[],"vendor":{"zero":0,"flag":false}},{"type":"function_call","id":"fc_page","call_id":"call_page","name":"newapi_skills_read","arguments":"{\"offset\":24576}"}]}`,
		},
		{
			name: "claude", format: types.RelayFormatClaude, historyKey: "messages", addedItems: 2,
			history:  `{"role":"assistant","content":[{"type":"thinking","thinking":"original-reasoning","signature":"original-signed"},{"type":"redacted_thinking","data":"encrypted-original"},{"type":"tool_use","id":"old","name":"pwsh","input":{}}],"vendor_counter":9007199254740993}, {"role":"user","content":[{"type":"tool_result","tool_use_id":"old","content":"original-result"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"YWJj"}},{"type":"document","source":{"type":"base64","media_type":"application/pdf","data":"YWJj"}},{"type":"text","text":"task"}],"vendor_zero":-0}`,
			system:   `,"system":[{"type":"text","text":"original-system","cache_control":{"type":"ephemeral"},"unknown":false}]`,
			response: `{"role":"assistant","content":[{"type":"thinking","thinking":"private-reasoning","signature":"private-signed","vendor":{"zero":0,"flag":false}},{"type":"tool_use","id":"call_page","name":"newapi_skills_read","input":{"offset":24576}}]}`,
		},
		{
			name: "gemini", format: types.RelayFormatGemini, historyKey: "contents", addedItems: 2,
			history:  `{"role":"model","parts":[{"thoughtSignature":"original-reasoning","functionCall":{"id":"old","name":"pwsh","args":{}}}],"vendor_counter":9007199254740993}, {"role":"user","parts":[{"functionResponse":{"id":"old","name":"pwsh","response":{"result":"original-result"}}},{"inlineData":{"mimeType":"image/png","data":"YWJj"}},{"fileData":{"mimeType":"application/pdf","fileUri":"gs://fixture/document"}},{"text":"task"}],"vendor_zero":-0}`,
			system:   `,"system_instruction":{"role":"user","parts":[{"text":"original-system","unknown":false}],"vendor_counter":9007199254740993}`,
			response: `{"candidates":[{"content":{"role":"model","parts":[{"thoughtSignature":"private-signed","functionCall":{"id":"call_page","name":"newapi_skills_read","args":{"offset":24576}},"vendor":{"zero":0,"flag":false}}]}}]}`,
		},
	}
}

func TestGatewaySkillRequestPreservesEightPagedRounds(t *testing.T) {
	t.Setenv("GATEWAY_SKILLS_DISCOVERY_MAX_OUTPUT_TOKENS", "4096")
	for _, fixture := range gatewaySkillRequestFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			original := fmt.Sprintf(`{"model":"test","%s":[%s],"temperature":0,"vendor":{"number":9007199254740993,"zero":-0,"flag":false}%s}`, fixture.historyKey, fixture.history, fixture.system)
			body := []byte(original)
			request, err := NewGatewaySkillRequest(body, fixture.format, "test", `{"skills":[]}`)
			require.NoError(t, err)
			assert.Equal(t, original, string(body))
			first := request.Bytes()
			firstSnapshot := string(first)
			initialItems := len(gjson.GetBytes(first, fixture.historyKey).Array())
			// 构造后客户端可以复用输入缓冲区，不能改变私有发现快照。
			body[0] = '!'
			assert.Equal(t, firstSnapshot, string(request.Bytes()))
			for round := range 8 {
				response := strings.ReplaceAll(fixture.response, "call_page", fmt.Sprintf("call_%d", round))
				turn, parseErr := ParseGatewaySkillTurn([]byte(response), fixture.format, nil)
				require.NoError(t, parseErr)
				require.Len(t, turn.Calls, 1)
				page := json.RawMessage(fmt.Sprintf(`{"content":"page_%d","next_offset":%d,"eof":%t}`, round, (round+1)*24576, round == 7))
				require.NoError(t, request.Continue(turn, []GatewaySkillResult{{Call: turn.Calls[0], Content: page}}))
				continued := request.Bytes()
				require.True(t, gjson.ValidBytes(continued))
				assert.Len(t, gjson.GetBytes(continued, fixture.historyKey).Array(), initialItems+(round+1)*fixture.addedItems)
				assert.Contains(t, string(continued), fixture.history, "原始多模态、签名和工具历史保持完整")
				assert.Contains(t, string(continued), fmt.Sprintf("page_%d", round))
				assert.Equal(t, "9007199254740993", gjson.GetBytes(continued, "vendor.number").Raw)
				assert.Equal(t, "-0", gjson.GetBytes(continued, "vendor.zero").Raw)
				assert.Equal(t, "false", gjson.GetBytes(continued, "vendor.flag").Raw)
				assert.Equal(t, "0", gjson.GetBytes(continued, "temperature").Raw)
				assert.Equal(t, firstSnapshot, string(first), "后续轮次不能改写已交给中继的正文")
			}
			beforeRefocus := request.Bytes()
			require.NoError(t, request.Refocus())
			refocused := request.Bytes()
			assert.Len(t, gjson.GetBytes(refocused, fixture.historyKey).Array(), initialItems+8*fixture.addedItems+1)
			assert.Contains(t, string(refocused), "Private gateway discovery correction")
			assert.Contains(t, string(refocused), fixture.history)
			for round := range 8 {
				assert.Contains(t, string(refocused), fmt.Sprintf("page_%d", round))
			}
			assert.NotContains(t, string(beforeRefocus), "Private gateway discovery correction")
			assert.Equal(t, firstSnapshot, string(first))
		})
	}
}

func TestGatewaySkillRequestContinuationFailureKeepsSnapshot(t *testing.T) {
	for _, fixture := range gatewaySkillRequestFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			request, err := NewGatewaySkillRequest([]byte(fmt.Sprintf(`{"%s":[%s]}`, fixture.historyKey, fixture.history)), fixture.format, "test", "")
			require.NoError(t, err)
			before := string(request.Bytes())
			turn, err := ParseGatewaySkillTurn([]byte(fixture.response), fixture.format, nil)
			require.NoError(t, err)
			wrong := GatewaySkillResult{Call: GatewaySkillCall{ID: "wrong", Name: GatewaySkillsReadTool}, Content: json.RawMessage(`{}`)}
			require.Error(t, request.Continue(turn, []GatewaySkillResult{wrong}))
			assert.Equal(t, before, string(request.Bytes()))
			if fixture.format == types.RelayFormatClaude || fixture.format == types.RelayFormatOpenAIResponses {
				turn.Output = []json.RawMessage{json.RawMessage(`{`)}
			} else {
				turn.Assistant = json.RawMessage(`{`)
			}
			require.Error(t, request.Continue(turn, []GatewaySkillResult{{Call: turn.Calls[0], Content: json.RawMessage(`{}`)}}))
			assert.Equal(t, before, string(request.Bytes()), "新增助手项编码失败必须保持此前可用状态")
		})
	}
}
