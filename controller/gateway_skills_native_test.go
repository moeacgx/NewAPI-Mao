package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 原生协议必须通过完整 helper 链验收，不能只验证私有工具 JSON 的构造函数。
func TestGatewaySkillsNativeProtocolsPreserveContinuationAndClientRequest(t *testing.T) {
	for _, protocol := range []struct {
		name        string
		format      types.RelayFormat
		channelType int
		route       string
	}{
		{name: "Responses", format: types.RelayFormatOpenAIResponses, channelType: constant.ChannelTypeOpenAI, route: "/v1/responses"},
		{name: "Claude", format: types.RelayFormatClaude, channelType: constant.ChannelTypeAnthropic, route: "/v1/messages"},
		{name: "Gemini", format: types.RelayFormatGemini, channelType: constant.ChannelTypeGemini, route: "/v1beta/models/*model"},
	} {
		for _, stream := range []bool{false, true} {
			for _, passthrough := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/stream=%t/passthrough=%t", protocol.name, stream, passthrough), func(t *testing.T) {
					var template *model.PromptTemplate
					calls := 0
					upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						body, err := io.ReadAll(r.Body)
						assert.NoError(t, err)
						calls++
						if calls > 3 {
							assert.Fail(t, "技能读取没有按预期结束", "unexpected upstream call %d", calls)
							http.Error(w, "unexpected extra call", http.StatusInternalServerError)
							return
						}
						assert.NotContains(t, string(body), "irrelevant-skill-must-stay-private")
						assert.NotContains(t, string(body), "unused-resource-must-stay-private")
						assert.NotContains(t, string(body), "reference-secret-only-after-read")
						assert.NotContains(t, string(body), "main-unread-tail")
						if calls < 3 {
							assert.NotContains(t, string(body), "client_lookup")
							assert.Contains(t, string(body), service.GatewaySkillsReadTool)
							assert.False(t, gjson.GetBytes(body, "stream").Bool())
							assert.NotContains(t, r.URL.Path, "streamGenerateContent")
							assert.Empty(t, r.URL.Query().Get("alt"))
							if calls == 1 {
								assert.NotContains(t, string(body), "main-secret-start")
							} else {
								assert.Contains(t, string(body), "main-secret-start")
								assertGatewaySkillNativeContinuation(t, protocol.format, body)
							}
							arguments := map[string]any{"template_id": template.ID, "skill_id": template.Skills[0].ID, "version": template.Version, "path": "SKILL.md"}
							writeGatewaySkillNativeResponse(t, w, protocol.format, false, calls == 1, arguments, 20, 5)
							return
						}
						assert.Contains(t, string(body), "main-secret-start")
						assert.NotContains(t, string(body), "newapi_skills")
						assert.NotContains(t, string(body), "SKILL_CONTEXT_READY")
						assert.NotContains(t, string(body), "private-encrypted-reasoning")
						assert.NotContains(t, string(body), "private-thinking-text")
						switch protocol.format {
						case types.RelayFormatOpenAIResponses:
							assert.Equal(t, "/v1/responses", r.URL.Path)
							assert.Equal(t, stream, gjson.GetBytes(body, "stream").Bool())
							assert.Equal(t, "client_lookup", gjson.GetBytes(body, "tools.0.name").String())
							assert.Equal(t, "required", gjson.GetBytes(body, "tool_choice").String())
							assert.Equal(t, "0", gjson.GetBytes(body, "temperature").Raw)
						case types.RelayFormatClaude:
							assert.Equal(t, "/v1/messages", r.URL.Path)
							assert.Equal(t, stream, gjson.GetBytes(body, "stream").Bool())
							assert.Equal(t, "client_lookup", gjson.GetBytes(body, "tools.0.name").String())
							assert.Equal(t, "auto", gjson.GetBytes(body, "tool_choice.type").String())
							assert.False(t, gjson.GetBytes(body, "tool_choice.disable_parallel_tool_use").Bool())
							assert.Equal(t, "enabled", gjson.GetBytes(body, "thinking.type").String())
						case types.RelayFormatGemini:
							assert.Equal(t, stream, strings.HasSuffix(r.URL.Path, ":streamGenerateContent"))
							assert.Equal(t, "client_lookup", gjson.GetBytes(body, "tools.0.functionDeclarations.0.name").String())
							assert.Equal(t, "ANY", gjson.GetBytes(body, "toolConfig.functionCallingConfig.mode").String())
							assert.Equal(t, "0", gjson.GetBytes(body, "generationConfig.temperature").Raw)
							if stream {
								assert.Equal(t, "sse", r.URL.Query().Get("alt"))
							}
						}
						writeGatewaySkillNativeResponse(t, w, protocol.format, stream, false, nil, 40, 10)
					}))
					defer upstream.Close()
					engine, user, token, saved := setupGatewaySkillsRelay(t, upstream.URL, passthrough, 10000)
					template = saved
					require.NoError(t, model.DB.Model(&model.Channel{}).Where("models = ?", canvasRouteBillingModel).Update("type", protocol.channelType).Error)
					model.InitChannelCache()
					engine.POST(protocol.route, func(c *gin.Context) { Relay(c, protocol.format) })
					path, body := gatewaySkillNativeClientRequest(t, protocol.format, stream)
					request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
					request.Header.Set("Content-Type", "application/json")
					request.Header.Set("Authorization", "Bearer sk-"+token.Key)
					request.Header.Set("anthropic-version", "2023-06-01")
					recorder := httptest.NewRecorder()
					engine.ServeHTTP(recorder, request)
					require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
					assert.Equal(t, 3, calls)
					assert.Contains(t, recorder.Body.String(), "final-answer")
					assert.NotContains(t, recorder.Body.String(), "newapi_skills")
					assert.NotContains(t, recorder.Body.String(), "SKILL_CONTEXT_READY")
					assert.NotContains(t, recorder.Body.String(), "private-encrypted-reasoning")
					assert.NotContains(t, recorder.Body.String(), "private-thinking-text")
					assert.NotContains(t, recorder.Body.String(), "main-secret-start")
					if stream {
						assert.Contains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
					}
					var storedUser model.User
					require.NoError(t, model.DB.First(&storedUser, user.Id).Error)
					var storedToken model.Token
					require.NoError(t, model.DB.First(&storedToken, token.Id).Error)
					assert.EqualValues(t, 9700, storedUser.Quota)
					assert.EqualValues(t, 9700, storedToken.RemainQuota)
					var logs []model.Log
					require.NoError(t, model.LOG_DB.Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).Order("id").Find(&logs).Error)
					require.Len(t, logs, 3)
					for index, log := range logs {
						assert.Equal(t, 100, log.Quota)
						assert.NotContains(t, log.Other, "main-secret-start")
						assert.NotContains(t, log.Other, "private-encrypted-reasoning")
						if index < 2 {
							assert.Equal(t, "discovery", gjson.Get(log.Other, "gateway_skills.stage").String())
							assert.Equal(t, 20, log.PromptTokens)
							assert.Equal(t, 5, log.CompletionTokens)
						} else {
							assert.Equal(t, "generation", gjson.Get(log.Other, "gateway_skills.stage").String())
							assert.Len(t, gjson.Get(log.Other, "gateway_skills.files").Array(), 1)
							assert.Equal(t, 40, log.PromptTokens)
							assert.Equal(t, 10, log.CompletionTokens)
						}
					}
				})
			}
		}
	}
}

func gatewaySkillNativeClientRequest(t *testing.T, format types.RelayFormat, stream bool) (string, string) {
	t.Helper()
	switch format {
	case types.RelayFormatOpenAIResponses:
		return "/v1/responses", fmt.Sprintf(`{"model":%q,"input":"review my code","instructions":"client-instructions","temperature":0,"stream":%t,"tools":[{"type":"function","name":"client_lookup","parameters":{"type":"object"}}],"tool_choice":"required"}`, canvasRouteBillingModel, stream)
	case types.RelayFormatClaude:
		return "/v1/messages", fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"review my code"}],"system":"client-instructions","max_tokens":4096,"thinking":{"type":"enabled","budget_tokens":1024},"stream":%t,"tools":[{"name":"client_lookup","description":"Look up client data","input_schema":{"type":"object"}}],"tool_choice":{"type":"auto","disable_parallel_tool_use":false}}`, canvasRouteBillingModel, stream)
	case types.RelayFormatGemini:
		action := "generateContent"
		if stream {
			action = "streamGenerateContent?alt=sse"
		}
		return "/v1beta/models/" + canvasRouteBillingModel + ":" + action, `{"contents":[{"role":"user","parts":[{"text":"review my code"}]}],"systemInstruction":{"parts":[{"text":"client-instructions"}]},"generationConfig":{"temperature":0},"tools":[{"functionDeclarations":[{"name":"client_lookup","description":"Look up client data","parameters":{"type":"object"}}]}],"toolConfig":{"functionCallingConfig":{"mode":"ANY","allowedFunctionNames":["client_lookup"]}}}`
	default:
		t.Fatalf("unsupported native protocol: %s", format)
		return "", ""
	}
}

func assertGatewaySkillNativeContinuation(t *testing.T, format types.RelayFormat, body []byte) {
	t.Helper()
	switch format {
	case types.RelayFormatOpenAIResponses:
		assert.True(t, gjson.GetBytes(body, "include").IsArray())
		assert.Contains(t, string(body), "reasoning.encrypted_content")
		assert.Equal(t, "private-encrypted-reasoning", gjson.GetBytes(body, "input.1.encrypted_content").String())
		assert.Equal(t, "reasoning", gjson.GetBytes(body, "input.1.type").String())
		assert.Equal(t, "call_read_1", gjson.GetBytes(body, "input.2.call_id").String())
		assert.Equal(t, "call_read_1", gjson.GetBytes(body, "input.3.call_id").String())
		assert.Equal(t, "function_call_output", gjson.GetBytes(body, "input.3.type").String())
	case types.RelayFormatClaude:
		assert.Equal(t, "thinking", gjson.GetBytes(body, "messages.1.content.0.type").String())
		assert.Equal(t, "private-thinking-text", gjson.GetBytes(body, "messages.1.content.0.thinking").String())
		assert.Equal(t, "c2lnbmF0dXJl", gjson.GetBytes(body, "messages.1.content.0.signature").String())
		assert.Equal(t, "private-redacted-data", gjson.GetBytes(body, "messages.1.content.1.data").String())
		assert.Equal(t, "call_read_1", gjson.GetBytes(body, "messages.1.content.2.id").String())
		assert.Equal(t, "call_read_1", gjson.GetBytes(body, "messages.2.content.0.tool_use_id").String())
	case types.RelayFormatGemini:
		assert.Equal(t, "c2lnbmF0dXJl", gjson.GetBytes(body, "contents.1.parts.0.thoughtSignature").String())
		assert.Equal(t, "call_read_1", gjson.GetBytes(body, "contents.1.parts.0.functionCall.id").String())
		assert.Equal(t, service.GatewaySkillsReadTool, gjson.GetBytes(body, "contents.1.parts.0.functionCall.name").String())
		assert.Equal(t, "call_read_1", gjson.GetBytes(body, "contents.2.parts.0.functionResponse.id").String())
		assert.Equal(t, service.GatewaySkillsReadTool, gjson.GetBytes(body, "contents.2.parts.0.functionResponse.name").String())
	}
}

func writeGatewaySkillNativeResponse(t *testing.T, w http.ResponseWriter, format types.RelayFormat, stream, read bool, arguments map[string]any, promptTokens, completionTokens int) {
	t.Helper()
	text := "SKILL_CONTEXT_READY"
	if promptTokens == 40 {
		text = "final-answer"
	}
	var response map[string]any
	switch format {
	case types.RelayFormatOpenAIResponses:
		output := []any{map[string]any{"type": "message", "id": "msg_final", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": text, "annotations": []any{}}}}}
		if read {
			encodedArguments, err := common.Marshal(arguments)
			require.NoError(t, err)
			output = []any{map[string]any{"type": "reasoning", "id": "reasoning_1", "summary": []any{}, "encrypted_content": "private-encrypted-reasoning"},
				map[string]any{"type": "function_call", "id": "fc_read_1", "call_id": "call_read_1", "name": service.GatewaySkillsReadTool, "arguments": string(encodedArguments), "status": "completed"}}
		}
		response = map[string]any{"id": "resp_native", "object": "response", "status": "completed", "model": canvasRouteBillingModel, "output": output,
			"usage": map[string]any{"input_tokens": promptTokens, "output_tokens": completionTokens, "total_tokens": promptTokens + completionTokens}}
	case types.RelayFormatClaude:
		content := []any{map[string]any{"type": "text", "text": text}}
		stopReason := "end_turn"
		if read {
			content = []any{map[string]any{"type": "thinking", "thinking": "private-thinking-text", "signature": "c2lnbmF0dXJl"},
				map[string]any{"type": "redacted_thinking", "data": "private-redacted-data"},
				map[string]any{"type": "tool_use", "id": "call_read_1", "name": service.GatewaySkillsReadTool, "input": arguments}}
			stopReason = "tool_use"
		}
		response = map[string]any{"id": "msg_native", "type": "message", "role": "assistant", "model": canvasRouteBillingModel, "content": content, "stop_reason": stopReason,
			"usage": map[string]any{"input_tokens": promptTokens, "output_tokens": completionTokens}}
	case types.RelayFormatGemini:
		parts := []any{map[string]any{"text": text}}
		if read {
			parts = []any{map[string]any{"functionCall": map[string]any{"id": "call_read_1", "name": service.GatewaySkillsReadTool, "args": arguments}, "thoughtSignature": "c2lnbmF0dXJl"}}
		}
		response = map[string]any{"responseId": "gemini_native", "modelVersion": canvasRouteBillingModel,
			"candidates":    []any{map[string]any{"index": 0, "content": map[string]any{"role": "model", "parts": parts}, "finishReason": "STOP"}},
			"usageMetadata": map[string]any{"promptTokenCount": promptTokens, "candidatesTokenCount": completionTokens, "totalTokenCount": promptTokens + completionTokens}}
	}
	encoded, err := common.Marshal(response)
	require.NoError(t, err)
	if !stream {
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(encoded)
		assert.NoError(t, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	switch format {
	case types.RelayFormatOpenAIResponses:
		_, err = io.WriteString(w, `event: response.output_text.delta`+"\n"+`data: {"type":"response.output_text.delta","output_index":0,"item_id":"msg_final","content_index":0,"delta":"final-answer"}`+"\n\n"+
			"event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":"+string(encoded)+"}\n\n")
	case types.RelayFormatClaude:
		_, err = io.WriteString(w, "event: message_start\ndata: "+fmt.Sprintf(`{"type":"message_start","message":{"id":"msg_native","type":"message","role":"assistant","model":%q,"content":[],"usage":{"input_tokens":%d,"output_tokens":1}}}`, canvasRouteBillingModel, promptTokens)+"\n\n"+
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"+
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"final-answer\"}}\n\n"+
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n"+
			"event: message_delta\ndata: "+fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":%d,"output_tokens":%d}}`, promptTokens, completionTokens)+"\n\n"+
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	case types.RelayFormatGemini:
		_, err = io.WriteString(w, "data: "+string(encoded)+"\n\n")
	}
	assert.NoError(t, err)
}
