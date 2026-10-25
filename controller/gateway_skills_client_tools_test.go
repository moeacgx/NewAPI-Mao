package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewaySkillsHandOffClientToolsAfterReading(t *testing.T) {
	for _, test := range []struct {
		name  string
		steps []string
		fail  bool
	}{
		{"read-then-client-tool", []string{"read", "client", "final"}, false},
		{"correct-premature-client-tool", []string{"client", "read", "ready", "final"}, false},
		{"mixed-read-and-client-tool", []string{"mixed", "final"}, false},
		{"correct-empty-read-with-client-tool", []string{"empty-mixed", "read", "ready", "final"}, false},
		{"reject-repeated-premature-client-tool", []string{"client", "client"}, true},
	} {
		for _, stream := range []bool{false, true} {
			for _, passthrough := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/stream=%t/passthrough=%t", test.name, stream, passthrough), func(t *testing.T) {
					var template *model.PromptTemplate
					var mainBytes int
					calls := 0
					upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						body, err := io.ReadAll(r.Body)
						assert.NoError(t, err)
						calls++
						if !assert.LessOrEqual(t, calls, len(test.steps)) {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						step := test.steps[calls-1]
						// 原客户端工具历史必须完整保留；私有误调用不能混入任何后续请求。
						assert.Contains(t, string(body), "historical_agent_call")
						assert.Contains(t, string(body), "historical-command-result")
						assert.NotContains(t, string(body), "private_agent_call")
						assert.NotContains(t, string(body), "private-must-not-run")
						if calls == 2 && (test.steps[0] == "client" || test.steps[0] == "empty-mixed") {
							assert.Contains(t, string(body), "Private gateway discovery correction")
						}
						message := map[string]any{"role": "assistant", "content": "SKILL_CONTEXT_READY"}
						toolCalls := make([]any, 0)
						if step == "read" || step == "mixed" || step == "empty-mixed" {
							offset := 0
							if step == "empty-mixed" {
								offset = mainBytes
							}
							args, marshalErr := common.Marshal(map[string]any{"template_id": template.ID, "skill_id": template.Skills[0].ID, "version": template.Version, "offset": offset})
							assert.NoError(t, marshalErr)
							toolCalls = append(toolCalls, map[string]any{"id": "read_main", "type": "function", "function": map[string]any{"name": service.GatewaySkillsReadTool, "arguments": string(args)}})
						}
						if step == "client" || step == "mixed" || step == "empty-mixed" {
							toolCalls = append(toolCalls, map[string]any{"id": "private_agent_call", "type": "function", "function": map[string]any{"name": "pwsh", "arguments": `{"command":"private-must-not-run"}`}})
						}
						if step == "final" {
							assert.Equal(t, "pwsh", gjson.GetBytes(body, "tools.0.function.name").String())
							assert.Equal(t, "required", gjson.GetBytes(body, "tool_choice").String())
							assert.Equal(t, stream, gjson.GetBytes(body, "stream").Bool())
							assert.Contains(t, string(body), "main-secret-start")
							assert.NotContains(t, string(body), service.GatewaySkillsReadTool)
							assert.NotContains(t, string(body), "Private gateway discovery correction")
							assert.NotContains(t, string(body), "SKILL_CONTEXT_READY")
							toolCalls = append(toolCalls, map[string]any{"id": "final_agent_call", "type": "function", "function": map[string]any{"name": "pwsh", "arguments": `{"command":"client-owned-command"}`}})
							if stream {
								w.Header().Set("Content-Type", "text/event-stream")
								_, _ = io.WriteString(w, `data: {"id":"final","object":"chat.completion.chunk","model":"`+canvasRouteBillingModel+`","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"final_agent_call","type":"function","function":{"name":"pwsh","arguments":"{\"command\":\"client-owned-command\"}"}}]},"finish_reason":null}]}`+"\n\n"+
									`data: {"id":"final","object":"chat.completion.chunk","model":"`+canvasRouteBillingModel+`","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":20,"completion_tokens":5,"total_tokens":25}}`+"\n\ndata: [DONE]\n\n")
								return
							}
						} else {
							assert.Equal(t, service.GatewaySkillsListTool, gjson.GetBytes(body, "tools.0.function.name").String())
							assert.False(t, gjson.GetBytes(body, "stream").Bool())
						}
						if len(toolCalls) > 0 {
							message["content"] = nil
							message["tool_calls"] = toolCalls
						}
						encoded, marshalErr := common.Marshal(map[string]any{"id": fmt.Sprintf("round-%d", calls), "object": "chat.completion", "model": canvasRouteBillingModel,
							"choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": "tool_calls"}},
							"usage":   map[string]int{"prompt_tokens": 20, "completion_tokens": 5, "total_tokens": 25}})
						assert.NoError(t, marshalErr)
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write(encoded)
					}))
					defer upstream.Close()
					engine, user, token, saved := setupGatewaySkillsRelay(t, upstream.URL, passthrough, 10000)
					template = saved
					main, err := model.ReadActiveTokenSkillFile(token.Id, "default", template.ID, template.Skills[0].ID, template.Version, "SKILL.md", 0, 4)
					require.NoError(t, err)
					mainBytes = main.TotalBytes
					body := fmt.Sprintf(`{"model":%q,"messages":[{"role":"system","content":"Continue the Agent task using pwsh when needed."},{"role":"user","content":"review my code"},{"role":"assistant","content":null,"tool_calls":[{"id":"historical_agent_call","type":"function","function":{"name":"pwsh","arguments":"{}"}}]},{"role":"tool","tool_call_id":"historical_agent_call","content":"historical-command-result"},{"role":"user","content":"Use the installed code review skill and continue."}],"stream":%t,"tools":[{"type":"function","function":{"name":"pwsh","parameters":{"type":"object","properties":{"command":{"type":"string"}}}}}],"tool_choice":"required"}`, canvasRouteBillingModel, stream)
					request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
					request.Header.Set("Content-Type", "application/json")
					request.Header.Set("Authorization", "Bearer sk-"+token.Key)
					recorder := httptest.NewRecorder()
					engine.ServeHTTP(recorder, request)
					if test.fail {
						assert.Equal(t, http.StatusBadGateway, recorder.Code, recorder.Body.String())
					} else {
						require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
						assert.Contains(t, recorder.Body.String(), "final_agent_call")
					}
					assert.Equal(t, len(test.steps), calls)
					assert.NotContains(t, recorder.Body.String(), "private_agent_call")
					assert.NotContains(t, recorder.Body.String(), "main-secret-start")
					wantQuota := 10000 - 100*len(test.steps)
					require.Eventually(t, func() bool {
						var storedUser model.User
						var storedToken model.Token
						return model.DB.First(&storedUser, user.Id).Error == nil && model.DB.First(&storedToken, token.Id).Error == nil &&
							storedUser.Quota == int64(wantQuota) && storedToken.RemainQuota == wantQuota
					}, 3*time.Second, 10*time.Millisecond)
					var logs []model.Log
					require.NoError(t, model.LOG_DB.Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).Order("id").Find(&logs).Error)
					require.Len(t, logs, len(test.steps))
					if !test.fail {
						last := logs[len(logs)-1]
						assert.Equal(t, "generation", gjson.Get(last.Other, "gateway_skills.stage").String())
						loadedPages := 0
						for _, file := range gjson.Get(last.Other, "gateway_skills.files").Array() {
							if file.Get("bytes").Int() > 0 {
								loadedPages++
								assert.Zero(t, file.Get("offset").Int())
							}
						}
						assert.Equal(t, 1, loadedPages)
					}
				})
			}
		}
	}
}

func TestGatewaySkillsResponsesUnrecognizedAgentToolsDoNotBlockGeneration(t *testing.T) {
	for _, test := range []struct {
		name          string
		repeatUnknown bool
		wantSkillRead bool
	}{
		{name: "retry-then-read", wantSkillRead: true},
		{name: "repeat-then-fallback", repeatUnknown: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var template *model.PromptTemplate
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				calls++
				switch calls {
				case 1:
					assert.Equal(t, "auto", gjson.GetBytes(body, "tool_choice").String())
					assert.Len(t, gjson.GetBytes(body, "tools").Array(), 3)
					assert.NotContains(t, string(body), "client_lookup")
					writeGatewaySkillResponsesClientCall(t, w, "agent_runtime_tool")
				case 2:
					assert.Contains(t, string(body), "Private gateway discovery correction")
					if test.repeatUnknown {
						writeGatewaySkillResponsesClientCall(t, w, "agent_runtime_tool")
						return
					}
					assert.NotContains(t, string(body), "main-secret-start")
					arguments := map[string]any{"template_id": template.ID, "skill_id": template.Skills[0].ID, "version": template.Version, "path": "SKILL.md"}
					writeGatewaySkillNativeResponse(t, w, types.RelayFormatOpenAIResponses, false, true, arguments, 20, 5)
				case 3:
					if test.wantSkillRead {
						assert.Contains(t, string(body), "main-secret-start")
						arguments := map[string]any{"template_id": template.ID, "skill_id": template.Skills[0].ID, "version": template.Version, "path": "SKILL.md", "offset": 24576}
						writeGatewaySkillNativeResponse(t, w, types.RelayFormatOpenAIResponses, false, true, arguments, 20, 5)
						return
					}
					assert.Contains(t, gjson.GetBytes(body, "input").Raw, "review this code")
					assert.Equal(t, "client_lookup", gjson.GetBytes(body, "tools.0.name").String())
					assert.Equal(t, "required", gjson.GetBytes(body, "tool_choice").String())
					assert.NotContains(t, string(body), "Private gateway discovery correction")
					assert.NotContains(t, string(body), service.GatewaySkillsReadTool)
					assert.NotContains(t, string(body), "main-secret-start")
					writeGatewaySkillNativeResponse(t, w, types.RelayFormatOpenAIResponses, false, false, nil, 40, 10)
				case 4:
					assert.Contains(t, string(body), "main-secret-start")
					writeGatewaySkillNativeResponse(t, w, types.RelayFormatOpenAIResponses, false, false, nil, 20, 5)
				case 5:
					assert.Contains(t, gjson.GetBytes(body, "input").Raw, "review this code")
					assert.Equal(t, "client_lookup", gjson.GetBytes(body, "tools.0.name").String())
					assert.Equal(t, "required", gjson.GetBytes(body, "tool_choice").String())
					assert.NotContains(t, string(body), "Private gateway discovery correction")
					assert.NotContains(t, string(body), service.GatewaySkillsReadTool)
					assert.Contains(t, gjson.GetBytes(body, "instructions").String(), "main-secret-start")
					writeGatewaySkillNativeResponse(t, w, types.RelayFormatOpenAIResponses, false, false, nil, 40, 10)
				default:
					assert.Fail(t, "unexpected upstream request", "call %d", calls)
					http.Error(w, "unexpected upstream request", http.StatusInternalServerError)
				}
			}))
			defer upstream.Close()

			engine, _, token, saved := setupGatewaySkillsRelay(t, upstream.URL, false, 10000)
			template = saved
			engine.POST("/v1/responses", func(c *gin.Context) { Relay(c, types.RelayFormatOpenAIResponses) })
			body := fmt.Sprintf(`{"model":%q,"input":"review this code","instructions":"The Agent may use agent_runtime_tool during normal generation.","tools":[{"type":"function","name":"client_lookup","parameters":{"type":"object"}}],"tool_choice":"required"}`, canvasRouteBillingModel)
			request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer sk-"+token.Key)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			wantCalls := 3
			if test.wantSkillRead {
				wantCalls = 5
			}
			assert.Equal(t, wantCalls, calls)
			assert.Equal(t, "completed", gjson.Get(recorder.Body.String(), "status").String())
			assert.Contains(t, recorder.Body.String(), "final-answer")
			assert.NotContains(t, recorder.Body.String(), "newapi_skills")

			var logs []model.Log
			require.NoError(t, model.LOG_DB.Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).Order("id").Find(&logs).Error)
			require.Len(t, logs, wantCalls)
			finalLog := logs[len(logs)-1]
			assert.Equal(t, "generation", gjson.Get(finalLog.Other, "gateway_skills.stage").String())
			assert.Equal(t, test.repeatUnknown, gjson.Get(finalLog.Other, "gateway_skills.discovery_fallback").Bool())
			files := gjson.Get(finalLog.Other, "gateway_skills.files").Array()
			if test.wantSkillRead {
				require.Len(t, files, 2)
				assert.EqualValues(t, 0, files[0].Get("offset").Int())
				assert.Positive(t, files[0].Get("bytes").Int())
				assert.EqualValues(t, files[0].Get("bytes").Int(), files[1].Get("offset").Int())
				assert.Positive(t, files[1].Get("bytes").Int())
			} else {
				assert.Empty(t, files)
			}
		})
	}
}

func writeGatewaySkillResponsesClientCall(t *testing.T, w http.ResponseWriter, name string) {
	t.Helper()
	encoded, err := common.Marshal(map[string]any{
		"id": "resp_runtime_tool", "object": "response", "status": "completed",
		"output": []any{map[string]any{"type": "function_call", "id": "fc_runtime_tool", "call_id": "call_runtime_tool", "name": name, "arguments": "{}", "status": "completed"}},
		"usage":  map[string]int{"input_tokens": 20, "output_tokens": 5, "total_tokens": 25},
	})
	require.NoError(t, err)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(encoded)
	assert.NoError(t, err)
}
