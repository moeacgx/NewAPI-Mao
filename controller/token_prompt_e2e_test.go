package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
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
			for _, playground := range []bool{false, true} {
				t.Run(fmt.Sprintf("stream=%t/passthrough=%t/playground=%t", stream, passthrough, playground), func(t *testing.T) {
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
					require.NoError(t, model.DB.AutoMigrate(&model.PromptCategory{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}))
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
					engine.Use(middleware.BodyStorageCleanup())
					path := "/v1/chat/completions"
					authorization := "sk-" + token.Key
					if playground {
						path = "/pg/chat/completions"
						authorization, _, err = service.IssueAccessToken(service.AuthIdentity{UserID: user.Id, SessionID: "canvas-route-" + user.Username, UserAuthVersion: user.AuthVersion, SessionVersion: 1})
						require.NoError(t, err)
						engine.Use(middleware.UserAuth(), middleware.PlaygroundTokenAuth(), middleware.PromptAudit(), middleware.Distribute())
						engine.POST(path, Playground)
					} else {
						engine.Use(middleware.TokenAuth(), middleware.PromptAudit(), middleware.Distribute())
						engine.POST(path, func(c *gin.Context) { Relay(c, types.RelayFormatOpenAI) })
					}
					body := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"question"}],"stream":%t}`, canvasRouteBillingModel, stream)
					request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
					request.Header.Set("Content-Type", "application/json")
					request.Header.Set("Authorization", "Bearer "+authorization)
					if playground {
						request.Header.Set("X-Playground-Token-Id", strconv.Itoa(token.Id))
					}
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
}

func TestPlaygroundTokenSelectionRestrictsModelsAndRequestedGroup(t *testing.T) {
	require.NoError(t, i18n.Init())
	setupCanvasRouteBillingTest(t, "http://127.0.0.1:1")
	model.InitDBColumns()
	user, _ := createCanvasRouteUserSession(t, "playground-limits", "wallet_only", 10000)
	accessToken, _, err := service.IssueAccessToken(service.AuthIdentity{UserID: user.Id, SessionID: "canvas-route-" + user.Username, UserAuthVersion: user.AuthVersion, SessionVersion: 1})
	require.NoError(t, err)
	tokens := []model.Token{
		{UserId: user.Id, Name: "allowed", Key: "playgroundallowed", Group: "default", Status: common.TokenStatusEnabled, ExpiredTime: -1, RemainQuota: 10000, ModelLimitsEnabled: true, ModelLimits: canvasRouteBillingModel},
		{UserId: user.Id, Name: "restricted", Key: "playgroundrestricted", Group: "default", Status: common.TokenStatusEnabled, ExpiredTime: -1, RemainQuota: 10000, ModelLimitsEnabled: true, ModelLimits: "different-model"},
	}
	require.NoError(t, model.DB.Create(&tokens).Error)
	engine := gin.New()
	engine.Use(middleware.BodyStorageCleanup(), middleware.UserAuth(), middleware.PlaygroundTokenAuth())
	engine.GET("/pg/token/models", func(c *gin.Context) { ListModels(c, constant.ChannelTypeOpenAI) })
	engine.POST("/pg/chat/completions", middleware.Distribute(), func(c *gin.Context) { t.Error("越权请求不得进入上游") })
	for _, test := range []struct {
		name       string
		tokenIndex int
		method     string
		path       string
		body       string
		status     int
		modelCount int64
	}{
		{name: "令牌A可用模型", method: http.MethodGet, path: "/pg/token/models", status: http.StatusOK, modelCount: 1},
		{name: "切换B清空模型", tokenIndex: 1, method: http.MethodGet, path: "/pg/token/models", status: http.StatusOK},
		{name: "B禁止使用A的模型", tokenIndex: 1, method: http.MethodPost, path: "/pg/chat/completions", body: fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"test"}]}`, canvasRouteBillingModel), status: http.StatusForbidden},
		{name: "绑定令牌不能切换自动分组", method: http.MethodPost, path: "/pg/chat/completions", body: fmt.Sprintf(`{"model":%q,"group":"auto","messages":[{"role":"user","content":"test"}]}`, canvasRouteBillingModel), status: http.StatusForbidden},
		{name: "返回A恢复模型", method: http.MethodGet, path: "/pg/token/models", status: http.StatusOK, modelCount: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Authorization", "Bearer "+accessToken)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Playground-Token-Id", strconv.Itoa(tokens[test.tokenIndex].Id))
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			require.Equal(t, test.status, response.Code, response.Body.String())
			assert.NotContains(t, response.Body.String(), tokens[test.tokenIndex].Key)
			if test.method == http.MethodGet {
				assert.True(t, gjson.GetBytes(response.Body.Bytes(), "success").Bool())
				assert.Equal(t, test.modelCount, gjson.GetBytes(response.Body.Bytes(), "data.#").Int())
			}
		})
	}
}
