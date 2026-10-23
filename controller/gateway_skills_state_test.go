package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 无法隔离的服务端会话必须在第一次上游调用前失败，不能污染原会话或产生消费。
func TestGatewaySkillsRejectServerStateBeforeUpstreamAndRefund(t *testing.T) {
	for _, test := range []struct {
		name    string
		format  types.RelayFormat
		route   string
		path    string
		request string
	}{
		{"responses-conversation-id", types.RelayFormatOpenAIResponses, "/v1/responses", "/v1/responses", `{"model":%q,"input":"review my code","conversation":"conv_original"}`},
		{"responses-conversation-object", types.RelayFormatOpenAIResponses, "/v1/responses", "/v1/responses", `{"model":%q,"input":"review my code","conversation":{"id":"conv_original"}}`},
		{"gemini-cached-content", types.RelayFormatGemini, "/v1beta/models/*model", "/v1beta/models/" + canvasRouteBillingModel + ":generateContent", `{"model":%q,"contents":[{"parts":[{"text":"review my code"}]}],"cachedContent":"cachedContents/original"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				http.Error(w, "must not call upstream", http.StatusInternalServerError)
			}))
			defer upstream.Close()
			engine, user, token, _ := setupGatewaySkillsRelay(t, upstream.URL, false, 10000)
			engine.POST(test.route, func(c *gin.Context) { Relay(c, test.format) })
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(fmt.Sprintf(test.request, canvasRouteBillingModel)))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer sk-"+token.Key)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
			assert.Contains(t, recorder.Body.String(), "invalid_request")
			assert.Zero(t, calls)
			var storedUser model.User
			require.NoError(t, model.DB.First(&storedUser, user.Id).Error)
			var storedToken model.Token
			require.NoError(t, model.DB.First(&storedToken, token.Id).Error)
			assert.EqualValues(t, 10000, storedUser.Quota)
			assert.EqualValues(t, 10000, storedToken.RemainQuota)
			var count int64
			require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("token_id = ? AND type = ?", token.Id, model.LogTypeConsume).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}
