package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func setupGatewaySkillsRelay(t *testing.T, upstreamURL string, passthrough bool, quota int) (*gin.Engine, *model.User, model.Token, *model.PromptTemplate) {
	t.Helper()
	setupCanvasRouteBillingTest(t, upstreamURL)
	model.InitDBColumns()
	require.NoError(t, model.DB.AutoMigrate(&model.PromptCategory{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{},
		&model.PromptSkillRecord{}, &model.PromptSkillFile{}, &model.PromptSkillChunk{}))
	user, _ := createCanvasRouteUserSession(t, "progressive-e2e", "wallet_only", quota)
	token := model.Token{UserId: user.Id, Name: "progressive", Key: "progressivee2ekey", Status: common.TokenStatusEnabled, ExpiredTime: -1, RemainQuota: quota}
	require.NoError(t, model.DB.Create(&token).Error)
	group, err := model.GetGroupByCodeOrAlias("default")
	require.NoError(t, err)
	template, err := model.SavePromptTemplate(model.PromptTemplate{Title: "技能合集", Kind: "bundle", Description: "测试按需读取", LoadMode: model.PromptLoadModeProgressive,
		Enabled: true, GroupIDs: []int{group.Id}, Skills: []model.PromptSkill{
			{Title: "代码审查", Description: "审查代码并参考相关规范", Path: "review/SKILL.md", Content: "main-secret-start\nRead references/rules.md when needed.\n" + strings.Repeat("code review instruction. ", 2000) + "\nmain-unread-tail",
				Resources: []model.PromptSkillResource{{Path: "references/rules.md", Content: "reference-secret-only-after-read"}, {Path: "references/unused.md", Content: "unused-resource-must-stay-private"}}},
			{Title: "翻译", Description: "仅适用于翻译任务", Path: "translate/SKILL.md", Content: "irrelevant-skill-must-stay-private"},
		}})
	require.NoError(t, err)
	require.NoError(t, model.ReplaceTokenPromptBindings(user.Id, token.Id, []model.TokenPromptBindingInput{{TemplateID: template.ID, Enabled: true}}, []int{group.Id}))
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("models = ?", canvasRouteBillingModel).Update("setting", fmt.Sprintf(`{"pass_through_body_enabled":%t}`, passthrough)).Error)
	model.InitChannelCache()
	service.InitHttpClient()
	engine := gin.New()
	engine.Use(middleware.BodyStorageCleanup(), middleware.TokenAuth(), middleware.PromptAudit(), middleware.Distribute())
	engine.POST("/v1/chat/completions", func(c *gin.Context) { Relay(c, types.RelayFormatOpenAI) })
	return engine, user, token, template
}

func TestGatewaySkillsReadProgressivelyAndPreserveClientTools(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, passthrough := range []bool{false, true} {
			t.Run(fmt.Sprintf("stream=%t/passthrough=%t", stream, passthrough), func(t *testing.T) {
				var template *model.PromptTemplate
				calls := 0
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					calls++
					assert.NotContains(t, string(body), "irrelevant-skill-must-stay-private")
					assert.NotContains(t, string(body), "unused-resource-must-stay-private")
					assert.NotContains(t, string(body), "main-unread-tail")
					if calls < 4 {
						assert.False(t, gjson.GetBytes(body, "stream").Bool())
						assert.NotContains(t, string(body), "client_lookup")
						w.Header().Set("Content-Type", "application/json")
						message := map[string]any{"role": "assistant", "content": "SKILL_CONTEXT_READY"}
						if calls == 1 || calls == 2 {
							args := map[string]any{"template_id": template.ID, "skill_id": template.Skills[0].ID, "version": template.Version, "path": "SKILL.md"}
							if calls == 2 {
								assert.Contains(t, string(body), "main-secret-start")
								assert.NotContains(t, string(body), "reference-secret-only-after-read")
								args["path"] = "references/rules.md"
							} else {
								assert.NotContains(t, string(body), "main-secret-start")
							}
							arguments, marshalErr := common.Marshal(args)
							assert.NoError(t, marshalErr)
							message = map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{map[string]any{"id": fmt.Sprintf("read_%d", calls), "type": "function",
								"function": map[string]any{"name": service.GatewaySkillsReadTool, "arguments": string(arguments)}}}}
						} else {
							assert.Contains(t, string(body), "reference-secret-only-after-read")
						}
						encoded, marshalErr := common.Marshal(map[string]any{"id": fmt.Sprintf("chatcmpl-skill-%d", calls), "object": "chat.completion", "model": canvasRouteBillingModel,
							"choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": "stop"}}, "usage": map[string]int{"prompt_tokens": 20, "completion_tokens": 5, "total_tokens": 25}})
						assert.NoError(t, marshalErr)
						_, _ = w.Write(encoded)
						return
					}
					assert.Equal(t, stream, gjson.GetBytes(body, "stream").Bool())
					assert.Contains(t, string(body), "main-secret-start")
					assert.Contains(t, string(body), "reference-secret-only-after-read")
					assert.Equal(t, "client_lookup", gjson.GetBytes(body, "tools.0.function.name").String())
					assert.Equal(t, "required", gjson.GetBytes(body, "tool_choice").String())
					assert.NotContains(t, string(body), "newapi_skills_read")
					assert.NotContains(t, string(body), "SKILL_CONTEXT_READY")
					assert.Equal(t, "0", gjson.GetBytes(body, "temperature").Raw)
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-final\",\"object\":\"chat.completion.chunk\",\"model\":\""+canvasRouteBillingModel+"\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"final-answer\"},\"finish_reason\":null}]}\n\n"+
							"data: {\"id\":\"chatcmpl-final\",\"object\":\"chat.completion.chunk\",\"model\":\""+canvasRouteBillingModel+"\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":40,\"completion_tokens\":10,\"total_tokens\":50}}\n\ndata: [DONE]\n\n")
					} else {
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, `{"id":"chatcmpl-final","object":"chat.completion","model":"`+canvasRouteBillingModel+`","choices":[{"index":0,"message":{"role":"assistant","content":"final-answer"},"finish_reason":"stop"}],"usage":{"prompt_tokens":40,"completion_tokens":10,"total_tokens":50}}`)
					}
				}))
				defer upstream.Close()
				engine, user, token, saved := setupGatewaySkillsRelay(t, upstream.URL, passthrough, 10000)
				template = saved
				body := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"review my code"}],"temperature":0,"stream":%t,"tools":[{"type":"function","function":{"name":"client_lookup","parameters":{"type":"object"}}}],"tool_choice":"required"}`, canvasRouteBillingModel, stream)
				request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", "Bearer sk-"+token.Key)
				recorder := httptest.NewRecorder()
				engine.ServeHTTP(recorder, request)
				require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
				assert.Equal(t, 4, calls)
				assert.Contains(t, recorder.Body.String(), "final-answer")
				assert.NotContains(t, recorder.Body.String(), "newapi_skills")
				assert.NotContains(t, recorder.Body.String(), "SKILL_CONTEXT_READY")
				var storedUser model.User
				require.NoError(t, model.DB.First(&storedUser, user.Id).Error)
				var storedToken model.Token
				require.NoError(t, model.DB.First(&storedToken, token.Id).Error)
				assert.EqualValues(t, 9600, storedUser.Quota)
				assert.EqualValues(t, 9600, storedToken.RemainQuota)
				var logs []model.Log
				require.NoError(t, model.LOG_DB.Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).Order("id").Find(&logs).Error)
				require.Len(t, logs, 4)
				for index, log := range logs {
					assert.Equal(t, 100, log.Quota)
					assert.NotContains(t, log.Other, "main-secret-start")
					assert.NotContains(t, log.Other, "reference-secret-only-after-read")
					if index < 3 {
						assert.Equal(t, "discovery", gjson.Get(log.Other, "gateway_skills.stage").String())
						assert.Equal(t, 20, log.PromptTokens)
					} else {
						assert.Equal(t, "generation", gjson.Get(log.Other, "gateway_skills.stage").String())
						assert.Len(t, gjson.Get(log.Other, "gateway_skills.files").Array(), 2)
						assert.Equal(t, 40, log.PromptTokens)
					}
				}
			})
		}
	}
}
