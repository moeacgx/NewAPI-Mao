package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordLoginIsIsolatedFromCriticalRateLimit(t *testing.T) {
	previousRedis := common.RedisEnabled
	previousGlobalEnabled, previousGlobalLimit, previousGlobalDuration := common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration
	previousCriticalEnabled, previousCriticalLimit, previousCriticalDuration := common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration
	previousLoginEnabled, previousLoginLimit, previousLoginIPLimit := common.LoginFailureRateLimitEnable, common.LoginFailureRateLimitNum, common.LoginFailureIPRateLimitNum
	previousLoginDuration, previousInflightLimit, previousLeaseDuration := common.LoginFailureRateLimitDuration, common.LoginInflightIPLimit, common.LoginInflightLeaseDuration
	previousPasswordLoginEnabled := common.PasswordLoginEnabled
	previousSessionSecret := common.SessionSecret
	common.RedisEnabled = false
	common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration = true, 100, 60
	common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = true, 1, 60
	common.LoginFailureRateLimitEnable, common.LoginFailureRateLimitNum, common.LoginFailureIPRateLimitNum = true, 5, 30
	common.LoginFailureRateLimitDuration, common.LoginInflightIPLimit, common.LoginInflightLeaseDuration = 60, 8, 10
	common.PasswordLoginEnabled = true
	common.SessionSecret = "router-login-rate-limit-secret"
	t.Cleanup(func() {
		common.RedisEnabled = previousRedis
		common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration = previousGlobalEnabled, previousGlobalLimit, previousGlobalDuration
		common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = previousCriticalEnabled, previousCriticalLimit, previousCriticalDuration
		common.LoginFailureRateLimitEnable, common.LoginFailureRateLimitNum, common.LoginFailureIPRateLimitNum = previousLoginEnabled, previousLoginLimit, previousLoginIPLimit
		common.LoginFailureRateLimitDuration, common.LoginInflightIPLimit, common.LoginInflightLeaseDuration = previousLoginDuration, previousInflightLimit, previousLeaseDuration
		common.PasswordLoginEnabled = previousPasswordLoginEnabled
		common.SessionSecret = previousSessionSecret
	})

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	SetApiRouter(engine)
	clientIP := "192.0.2.183:12345"

	logout := httptest.NewRequest(http.MethodPost, "/api/user/auth/logout", nil)
	logout.RemoteAddr = clientIP
	logoutResponse := httptest.NewRecorder()
	engine.ServeHTTP(logoutResponse, logout)
	require.NotEqual(t, http.StatusTooManyRequests, logoutResponse.Code)

	secondLogout := httptest.NewRequest(http.MethodPost, "/api/user/auth/logout", nil)
	secondLogout.RemoteAddr = clientIP
	secondLogoutResponse := httptest.NewRecorder()
	engine.ServeHTTP(secondLogoutResponse, secondLogout)
	require.Equal(t, http.StatusTooManyRequests, secondLogoutResponse.Code)

	login := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(`{}`))
	login.Header.Set("Content-Type", "application/json")
	login.RemoteAddr = clientIP
	loginResponse := httptest.NewRecorder()
	engine.ServeHTTP(loginResponse, login)
	assert.Equal(t, http.StatusOK, loginResponse.Code)
	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(loginResponse.Body.Bytes(), &response))
	assert.False(t, response.Success)
}
