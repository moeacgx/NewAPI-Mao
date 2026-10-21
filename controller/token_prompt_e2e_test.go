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

func TestTokenPromptAuthenticatedRelaySettlesUsageAndAudit(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, passthrough := range []bool{false, true} {
			t.Run(fmt.Sprintf("stream=%t/passthrough=%t", stream, passthrough), func(t *testing.T) {
				outbound := make(chan []byte, 1)
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					outbound <- body
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-prompt\",\"object\":\"chat.completion.chunk\",\"model\":\"canvas-route-priced-image\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"answer\"},\"finish_reason\":null}]}\n\n"+
							"data: {\"id\":\"chatcmpl-prompt\",\"object\":\"chat.completion.chunk\",\"model\":\"canvas-route-priced-image\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":20,\"completion_tokens\":5,\"total_tokens\":25}}\n\ndata: [DONE]\n\n")
					} else {
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, `{"id":"chatcmpl-prompt","object":"chat.completion","model":"canvas-route-priced-image","choices":[{"index":0,"message":{"role":"assistant","content":"answer"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":5,"total_tokens":25}}`)
					}
				}))
				defer upstream.Close()
				setupCanvasRouteBillingTest(t, upstream.URL)
				model.InitDBColumns()
				require.NoError(t, model.DB.AutoMigrate(&model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}))
				user, _ := createCanvasRouteUserSession(t, "prompt-end-to-end", "wallet_only", 10000)
				token := model.Token{UserId: user.Id, Name: "prompt-e2e", Key: "prompte2ekey", Status: common.TokenStatusEnabled, ExpiredTime: -1, RemainQuota: 10000}
				require.NoError(t, model.DB.Create(&token).Error)
				group, err := model.GetGroupByCodeOrAlias("default")
				require.NoError(t, err)
				template, err := model.SavePromptTemplate(model.PromptTemplate{Title: "技能", Content: "installed e2e instruction", Enabled: true, GroupIDs: []int{group.Id}})
				require.NoError(t, err)
				require.NoError(t, model.ReplaceTokenPromptBindings(user.Id, token.Id, []model.TokenPromptBindingInput{{TemplateID: template.ID, Enabled: true}}, []int{group.Id}))
				require.NoError(t, model.DB.Model(&model.Channel{}).Where("models = ?", canvasRouteBillingModel).Update("setting", fmt.Sprintf(`{"pass_through_body_enabled":%t}`, passthrough)).Error)
				model.InitChannelCache()
				service.InitHttpClient()
				engine := gin.New()
				engine.Use(middleware.BodyStorageCleanup(), middleware.TokenAuth(), middleware.PromptAudit(), middleware.Distribute())
				engine.POST("/v1/chat/completions", func(c *gin.Context) { Relay(c, types.RelayFormatOpenAI) })
				body := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"question"}],"stream":%t}`, canvasRouteBillingModel, stream)
				request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", "Bearer sk-"+token.Key)
				recorder := httptest.NewRecorder()
				engine.ServeHTTP(recorder, request)
				require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
				assert.Contains(t, recorder.Body.String(), "answer")
				assert.Equal(t, "installed e2e instruction", gjson.GetBytes(<-outbound, "messages.0.content").String())
				var storedUser model.User
				require.NoError(t, model.DB.First(&storedUser, user.Id).Error)
				var storedToken model.Token
				require.NoError(t, model.DB.First(&storedToken, token.Id).Error)
				assert.Equal(t, int64(9900), storedUser.Quota)
				assert.Equal(t, 9900, storedToken.RemainQuota)
				var log model.Log
				require.NoError(t, model.LOG_DB.Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).First(&log).Error)
				assert.Equal(t, 20, log.PromptTokens)
				assert.Equal(t, 5, log.CompletionTokens)
				assert.Equal(t, 100, log.Quota)
				assert.EqualValues(t, template.ID, gjson.Get(log.Other, "prompt_templates.0.id").Int())
				assert.NotContains(t, log.Other, "installed e2e instruction")
			})
		}
	}
}
