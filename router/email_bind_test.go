package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailBindRouteKeepsAuthenticatedAccount(t *testing.T) {
	for _, test := range []struct {
		name, body        string
		anonymous, replay bool
		wantStatus        int
		wantSuccess       bool
	}{
		{"匿名不能绑定", `{"email":"bind+identity@example.com","code":"123456"}`, true, false, http.StatusUnauthorized, false},
		{"错误验证码不修改账户", `{"email":"bind+identity@example.com","code":"654321"}`, false, false, http.StatusOK, false},
		{"非法JSON不修改账户", `{`, false, false, http.StatusOK, false},
		{"忽略伪造ID并绑定当前用户", `{"email":" Bind+Identity@Example.COM ","code":"123456","id":99999,"user_id":99999}`, false, false, http.StatusOK, true},
		{"验证码不能重放", `{"email":"bind+identity@example.com","code":"123456"}`, false, true, http.StatusOK, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			setupFeatureRouterAuthTest(t)
			require.NoError(t, model.DB.AutoMigrate(&model.AffiliateRecord{}))
			oldGlobal, oldCritical := common.GlobalApiRateLimitEnable, common.CriticalRateLimitEnable
			common.GlobalApiRateLimitEnable, common.CriticalRateLimitEnable = false, false
			t.Cleanup(func() { common.GlobalApiRateLimitEnable, common.CriticalRateLimitEnable = oldGlobal, oldCritical })
			user, accessToken := issueFeatureRouterSession(t, common.RoleCommonUser, "email-bind")
			other, _ := issueFeatureRouterSession(t, common.RoleCommonUser, "email-other")
			user.Quota, user.UsedQuota = 12345, 345
			require.NoError(t, model.DB.Model(user).Updates(map[string]any{"quota": user.Quota, "used_quota": user.UsedQuota}).Error)
			const email, code = "bind+identity@example.com", "123456"
			common.RegisterVerificationCodeWithKey(email, code, common.EmailVerificationPurpose)
			t.Cleanup(func() { common.DeleteKey(email, common.EmailVerificationPurpose) })
			engine := gin.New()
			SetApiRouter(engine)
			credential := accessToken
			if test.anonymous {
				credential = ""
			}
			if test.replay {
				first := requestEmailBinding(engine, accessToken, test.body)
				require.JSONEq(t, `{"success":true,"message":""}`, first.Body.String())
				user.Email = email
			}

			response := requestEmailBinding(engine, credential, test.body)
			require.Equal(t, test.wantStatus, response.Code, response.Body.String())
			var result struct {
				Success bool `json:"success"`
				Data    any  `json:"data"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			assert.Equal(t, test.wantSuccess, result.Success)
			assert.Nil(t, result.Data, "绑定不能返回新的登录身份或凭证")
			assert.Empty(t, response.Header().Values("Set-Cookie"))
			if test.wantSuccess {
				user.Email = email
			}
			var stored model.User
			require.NoError(t, model.DB.First(&stored, user.Id).Error)
			assert.Equal(t, *user, stored)
			var otherStored model.User
			require.NoError(t, model.DB.First(&otherStored, other.Id).Error)
			assert.Equal(t, *other, otherStored)
			var users, sessions int64
			require.NoError(t, model.DB.Unscoped().Model(&model.User{}).Count(&users).Error)
			require.NoError(t, model.DB.Model(&model.UserSession{}).Count(&sessions).Error)
			assert.EqualValues(t, 2, users)
			assert.EqualValues(t, 2, sessions)

			// 绑定前后的原访问令牌都对应原账号，未注册新用户或切换会话。
			self := serveFeatureRouterRequest(engine, http.MethodGet, "/api/user/self", accessToken)
			require.Equal(t, http.StatusOK, self.Code)
			var selfResult struct {
				Success bool       `json:"success"`
				Data    model.User `json:"data"`
			}
			require.NoError(t, common.Unmarshal(self.Body.Bytes(), &selfResult))
			assert.True(t, selfResult.Success)
			assert.Equal(t, user.Id, selfResult.Data.Id)
			assert.Equal(t, user.Email, selfResult.Data.Email)
			assert.Equal(t, user.Quota, selfResult.Data.Quota)
		})
	}
}

func requestEmailBinding(engine *gin.Engine, credential, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/oauth/email/bind", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}
