package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewaySkillsUnexpectedChatStreamPreservesReadingAndBilling(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, ending := range []string{"done", "finished-eof", "truncated"} {
			t.Run(fmt.Sprintf("stream=%t/ending=%s", stream, ending), func(t *testing.T) {
				truncated := ending == "truncated"
				var template *model.PromptTemplate
				calls := 0
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					calls++
					if calls == 1 {
						assert.False(t, gjson.GetBytes(body, "stream").Bool())
						arguments, err := common.Marshal(map[string]any{"template_id": template.ID, "skill_id": template.Skills[0].ID, "version": template.Version, "path": "SKILL.md"})
						assert.NoError(t, err)
						w.Header().Set("Content-Type", "text/event-stream")
						// 模拟供应商忽略非流式请求，并将工具参数拆成两段。
						for index, part := range []string{string(arguments[:len(arguments)/2]), string(arguments[len(arguments)/2:])} {
							tool := map[string]any{"index": 0, "function": map[string]any{"arguments": part}}
							delta := map[string]any{"tool_calls": []any{tool}}
							if index == 0 {
								tool["id"], tool["type"] = "read_stream", "function"
								tool["function"].(map[string]any)["name"] = service.GatewaySkillsReadTool
								delta["role"], delta["reasoning_content"] = "assistant", "private-stream-reasoning"
							}
							encoded, err := common.Marshal(map[string]any{"id": "chat-private", "object": "chat.completion.chunk", "model": canvasRouteBillingModel, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
							assert.NoError(t, err)
							_, err = fmt.Fprintf(w, "data: %s\n\n", encoded)
							assert.NoError(t, err)
						}
						if !truncated {
							_, err = io.WriteString(w, `data: {"id":"chat-private","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":20,"completion_tokens":5,"total_tokens":25}}`+"\n\n")
							assert.NoError(t, err)
							if ending == "done" {
								_, err = io.WriteString(w, "data: [DONE]\n\n")
								assert.NoError(t, err)
							}
						}
						return
					}
					assert.False(t, truncated, "截断发现阶段不能进入下一轮或正式生成")
					assert.Contains(t, string(body), "main-secret-start")
					if calls == 2 {
						assert.False(t, gjson.GetBytes(body, "stream").Bool())
						assert.Contains(t, string(body), "private-stream-reasoning")
						assert.Contains(t, string(body), "read_stream")
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"SKILL_CONTEXT_READY"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":5,"total_tokens":25}}`)
						return
					}
					assert.Equal(t, 3, calls)
					assert.Equal(t, stream, gjson.GetBytes(body, "stream").Bool())
					assert.Equal(t, "client_lookup", gjson.GetBytes(body, "tools.0.function.name").String())
					assert.Equal(t, "required", gjson.GetBytes(body, "tool_choice").String())
					assert.NotContains(t, string(body), "private-stream-reasoning")
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = io.WriteString(w, `data: {"choices":[{"index":0,"delta":{"role":"assistant","content":"final-answer"},"finish_reason":"stop"}],"usage":{"prompt_tokens":40,"completion_tokens":10,"total_tokens":50}}`+"\n\ndata: [DONE]\n\n")
					} else {
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"final-answer"},"finish_reason":"stop"}],"usage":{"prompt_tokens":40,"completion_tokens":10,"total_tokens":50}}`)
					}
				}))
				defer upstream.Close()
				engine, user, token, saved := setupGatewaySkillsRelay(t, upstream.URL, false, 10000)
				template = saved
				body := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"review code"}],"stream":%t,"tools":[{"type":"function","function":{"name":"client_lookup","parameters":{"type":"object"}}}],"tool_choice":"required"}`, canvasRouteBillingModel, stream)
				request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", "Bearer sk-"+token.Key)
				request.Header.Set(service.GatewaySkillsDisplayHeader, "metadata")
				recorder := httptest.NewRecorder()
				engine.ServeHTTP(recorder, request)
				var logs []model.Log
				require.NoError(t, model.LOG_DB.Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).Order("id").Find(&logs).Error)
				if truncated {
					require.Equal(t, http.StatusBadGateway, recorder.Code, recorder.Body.String())
					assert.Equal(t, 1, calls)
					require.Len(t, logs, 1)
					assert.Empty(t, gjson.Get(logs[0].Other, "gateway_skills.files").Array())
				} else {
					require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
					assert.Equal(t, 3, calls)
					require.Len(t, logs, 3)
					assert.Len(t, gjson.Get(logs[2].Other, "gateway_skills.files").Array(), 1)
					assert.Equal(t, "generation", gjson.Get(logs[2].Other, "gateway_skills.stage").String())
					assert.Contains(t, recorder.Body.String(), "final-answer")
					if stream {
						assert.Contains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
						assert.Contains(t, recorder.Body.String(), "[DONE]")
					}
				}
				assert.NotContains(t, recorder.Body.String(), "private-stream-reasoning")
				assert.NotContains(t, recorder.Body.String(), "main-secret-start")
				assert.NotContains(t, recorder.Body.String(), "SKILL_CONTEXT_READY")
				var storedUser model.User
				var storedToken model.Token
				require.NoError(t, model.DB.First(&storedUser, user.Id).Error)
				require.NoError(t, model.DB.First(&storedToken, token.Id).Error)
				assert.EqualValues(t, 10000-100*len(logs), storedUser.Quota)
				assert.EqualValues(t, 10000-100*len(logs), storedToken.RemainQuota)
			})
		}
	}
}
