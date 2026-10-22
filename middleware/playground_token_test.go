package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaygroundTokenAuthOwnershipAndRestrictions(t *testing.T) {
	for _, test := range []struct {
		name   string
		header string
		modify func(*model.Token)
		status int
	}{
		{name: "本人令牌", status: http.StatusOK},
		{name: "他人令牌", modify: func(token *model.Token) { token.UserId++ }, status: http.StatusNotFound},
		{name: "禁用令牌", modify: func(token *model.Token) { token.Status = common.TokenStatusDisabled }, status: http.StatusUnauthorized},
		{name: "过期令牌", modify: func(token *model.Token) { token.ExpiredTime = time.Now().Unix() - 60 }, status: http.StatusUnauthorized},
		{name: "耗尽令牌", modify: func(token *model.Token) { token.RemainQuota = 0 }, status: http.StatusUnauthorized},
		{name: "IP限制", modify: func(token *model.Token) { ips := "192.0.2.1"; token.AllowIps = &ips }, status: http.StatusForbidden},
		{name: "错误ID", header: "invalid", status: http.StatusBadRequest},
		{name: "零ID", header: "0", status: http.StatusBadRequest},
		{name: "不存在ID", header: "999999", status: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupDashboardAuthMiddlewareTest(t)
			model.InitDBColumns()
			require.NoError(t, model.DB.AutoMigrate(&model.Token{}))
			user := createMiddlewarePATUser(t, "playground-owner", "unused-pat")
			token := model.Token{UserId: user.Id, Name: "selected", Key: "playgroundselected", Status: common.TokenStatusEnabled, ExpiredTime: -1, RemainQuota: 1000}
			if test.modify != nil {
				test.modify(&token)
			}
			require.NoError(t, model.DB.Create(&token).Error)
			engine := gin.New()
			engine.GET("/pg/token/models", func(c *gin.Context) { c.Set("id", user.Id) }, PlaygroundTokenAuth(), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"token_id": c.GetInt("token_id"), "user_id": c.GetInt("id")})
			})
			request := httptest.NewRequest(http.MethodGet, "/pg/token/models", nil)
			request.RemoteAddr = "198.51.100.10:1234"
			value := test.header
			if value == "" {
				value = strconv.Itoa(token.Id)
			}
			request.Header.Set("X-Playground-Token-Id", value)
			// 客户端不能通过备用鉴权头覆盖已选择的令牌。
			request.Header.Set("Sec-WebSocket-Protocol", "openai-insecure-api-key.sk-invalid")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			require.Equal(t, test.status, response.Code, response.Body.String())
			assert.NotContains(t, response.Body.String(), token.Key)
			if test.status == http.StatusOK {
				var body struct {
					TokenID int `json:"token_id"`
					UserID  int `json:"user_id"`
				}
				require.NoError(t, common.Unmarshal(response.Body.Bytes(), &body))
				assert.Equal(t, token.Id, body.TokenID)
				assert.Equal(t, user.Id, body.UserID)
			}
		})
	}
}

func TestPlaygroundRejectsInvalidSelectionWithoutFallback(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []string
		userID int
		pat    bool
		status int
	}{
		{name: "空选择", values: []string{""}, userID: 1, status: http.StatusBadRequest},
		{name: "重复选择", values: []string{"1", "2"}, userID: 1, status: http.StatusBadRequest},
		{name: "未登录", values: []string{"1"}, status: http.StatusUnauthorized},
		{name: "个人访问令牌", values: []string{"1"}, userID: 1, pat: true, status: http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine := gin.New()
			engine.POST("/pg/chat/completions", func(c *gin.Context) {
				c.Set("id", test.userID)
				c.Set("use_access_token", test.pat)
			}, PlaygroundTokenAuth(), func(c *gin.Context) { t.Error("无效选择不得进入账号模式") })
			request := httptest.NewRequest(http.MethodPost, "/pg/chat/completions", nil)
			request.Header[http.CanonicalHeaderKey("X-Playground-Token-Id")] = test.values
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			assert.Equal(t, test.status, response.Code)
		})
	}
}

func TestPlaygroundWithoutTokenKeepsAccountMode(t *testing.T) {
	engine := gin.New()
	engine.POST("/pg/chat/completions", PlaygroundTokenAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"token_id": c.GetInt("token_id")})
	})
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/pg/chat/completions", nil))
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"token_id":0}`, response.Body.String())
}
