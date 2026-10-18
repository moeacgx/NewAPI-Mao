package router

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var tokenKeyCopyUserSequence atomic.Int64

func setupTokenKeyCopyRouter(t *testing.T, useRedis bool, maxReads, globalLimit int) (*gin.Engine, *miniredis.Miniredis) {
	t.Helper()
	t.Cleanup(model.InitDBColumns)
	setupFeatureRouterAuthTest(t)
	model.InitDBColumns()
	require.NoError(t, model.DB.AutoMigrate(&model.Token{}, &model.TokenGroupBinding{}))
	oldEnable, oldNum, oldDuration := common.TokenKeyReadRateLimitEnable, common.TokenKeyReadRateLimitNum, common.TokenKeyReadRateLimitDuration
	common.TokenKeyReadRateLimitEnable, common.TokenKeyReadRateLimitNum, common.TokenKeyReadRateLimitDuration = true, maxReads, 30
	t.Cleanup(func() {
		common.TokenKeyReadRateLimitEnable, common.TokenKeyReadRateLimitNum, common.TokenKeyReadRateLimitDuration = oldEnable, oldNum, oldDuration
	})
	return setupAuthRefreshRateLimitRouter(t, useRedis, globalLimit)
}

func newTokenKeyCopyUser(t *testing.T) (*model.User, string, *model.Token) {
	t.Helper()
	id := int(tokenKeyCopyUserSequence.Add(1)) + 10000000
	credential := fmt.Sprintf("copy-test-pat-%d", id)
	user := &model.User{Id: id, Username: fmt.Sprintf("copy-user-%d", id), Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AccessToken: &credential,
		AffCode: fmt.Sprintf("copy-aff-%d", id), AuthVersion: 1}
	require.NoError(t, model.DB.Create(user).Error)
	token := &model.Token{UserId: id, Key: fmt.Sprintf("copy-model-key-%d", id), Name: "copy-key", GroupMode: model.TokenGroupModeInherit}
	require.NoError(t, model.DB.Create(token).Error)
	return user, credential, token
}

func requestTokenKeyCopy(engine *gin.Engine, path, credential, ip, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.RemoteAddr = net.JoinHostPort(ip, "12345")
	request.Header.Set("Content-Type", "application/json")
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}

func TestTokenKeyCopyFirstReadAfterSharedCriticalLimitExhausted(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			engine, _ := setupTokenKeyCopyRouter(t, useRedis, 2, 100)
			user, credential, token := newTokenKeyCopyUser(t)
			ip := fmt.Sprintf("2001:db8:%x::%x", user.Id>>16, user.Id&0xffff)
			require.Equal(t, http.StatusOK, requestAuthRefreshRateLimit(engine, "/api/user/auth/logout", ip).Code)
			require.Equal(t, http.StatusTooManyRequests, requestAuthRefreshRateLimit(engine, "/api/user/auth/logout", ip).Code)
			response := requestTokenKeyCopy(engine, fmt.Sprintf("/api/token/%d/key", token.Id), credential, ip, "")
			require.Equal(t, http.StatusOK, response.Code, "首次复制不应被登录或其他 CT 操作拦截")
			assert.Contains(t, response.Body.String(), token.GetFullKey())
			assert.Contains(t, response.Header().Get("Cache-Control"), "no-store")
		})
	}
}

func TestTokenKeyCopyQuotaUsesUserAcrossIPsCredentialsAndReadRoutes(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			engine, redisServer := setupTokenKeyCopyRouter(t, useRedis, 2, 100)
			user, credential, token := newTokenKeyCopyUser(t)
			_, otherCredential, otherToken := newTokenKeyCopyUser(t)
			ip := fmt.Sprintf("2001:db8:%x::%x", user.Id>>16, user.Id&0xffff)
			otherIP := fmt.Sprintf("2001:db8:%x:1::%x", user.Id>>16, user.Id&0xffff)
			path := fmt.Sprintf("/api/token/%d/key", token.Id)
			session := &model.UserSession{SID: fmt.Sprintf("copy-session-%d", user.Id), UserID: user.Id, Version: 1,
				UserAuthVersion: 1, Status: model.UserSessionStatusActive, ExpiresAt: time.Now().Unix() + 3600,
				RefreshHash: fmt.Sprintf("copy-refresh-%d", user.Id), LoginMethod: "password"}
			require.NoError(t, model.CreateUserSession(session))
			sessionCredential, _, err := service.IssueAccessToken(service.AuthIdentity{UserID: user.Id, SessionID: session.SID, UserAuthVersion: 1, SessionVersion: 1})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, requestTokenKeyCopy(engine, path, credential, ip, "").Code)
			batch := requestTokenKeyCopy(engine, "/api/token/batch/keys", sessionCredential, otherIP, fmt.Sprintf(`{"ids":[%d]}`, token.Id))
			require.Equal(t, http.StatusOK, batch.Code)
			assert.Contains(t, batch.Body.String(), token.GetFullKey())
			blocked := requestTokenKeyCopy(engine, path, credential, otherIP, "")
			assert.Equal(t, http.StatusTooManyRequests, blocked.Code)
			assert.Equal(t, "30", blocked.Header().Get("Retry-After"))
			assert.Contains(t, blocked.Header().Get("Cache-Control"), "no-store")
			assert.Empty(t, blocked.Body.String())
			// 同出口另一位用户可读取自己的密钥，取密钥也不能挤占登录/注销额度。
			require.Equal(t, http.StatusOK, requestTokenKeyCopy(engine, fmt.Sprintf("/api/token/%d/key", otherToken.Id), otherCredential, ip, "").Code)
			require.Equal(t, http.StatusOK, requestAuthRefreshRateLimit(engine, "/api/user/auth/logout", ip).Code)
			if redisServer != nil {
				redisServer.FastForward(30 * time.Second)
				require.Equal(t, http.StatusOK, requestTokenKeyCopy(engine, path, credential, ip, "").Code)
			}
		})
	}
}

func TestTokenKeyCopyStillRequiresAuthenticationOwnershipAndBatchBound(t *testing.T) {
	engine, _ := setupTokenKeyCopyRouter(t, true, 20, 100)
	user, credential, token := newTokenKeyCopyUser(t)
	_, _, foreignToken := newTokenKeyCopyUser(t)
	ip := fmt.Sprintf("2001:db8:%x::%x", user.Id>>16, user.Id&0xffff)
	path := fmt.Sprintf("/api/token/%d/key", foreignToken.Id)
	for _, invalid := range []string{"", "invalid-pat"} {
		response := requestTokenKeyCopy(engine, path, invalid, ip, "")
		assert.Equal(t, http.StatusUnauthorized, response.Code)
		assert.NotContains(t, response.Body.String(), foreignToken.GetFullKey())
	}
	foreign := requestTokenKeyCopy(engine, path, credential, ip, "")
	assert.Contains(t, foreign.Body.String(), `"success":false`)
	assert.NotContains(t, foreign.Body.String(), foreignToken.GetFullKey())
	batch := requestTokenKeyCopy(engine, "/api/token/batch/keys", credential, ip, fmt.Sprintf(`{"ids":[%d,%d]}`, token.Id, foreignToken.Id))
	assert.Contains(t, batch.Body.String(), token.GetFullKey())
	assert.NotContains(t, batch.Body.String(), foreignToken.GetFullKey())
	ids := make([]int, 101)
	for i := range ids {
		ids[i] = token.Id
	}
	body, err := common.Marshal(map[string]any{"ids": ids})
	require.NoError(t, err)
	tooMany := requestTokenKeyCopy(engine, "/api/token/batch/keys", credential, ip, string(body))
	assert.Contains(t, tooMany.Body.String(), `"success":false`)
	assert.NotContains(t, tooMany.Body.String(), token.GetFullKey())
}

func TestTokenKeyCopyLimiterSwitchesDoNotDisableGlobalProtection(t *testing.T) {
	for _, test := range []struct {
		name                             string
		keyLimitEnabled, criticalEnabled bool
		globalLimit                      int
		wantSecond                       int
	}{
		{"取密钥开关独立", false, true, 100, http.StatusOK},
		{"关闭CT不关闭密钥保护", true, false, 100, http.StatusTooManyRequests},
		{"关闭密钥限流仍受GA限制", false, true, 1, http.StatusTooManyRequests},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _ = setupTokenKeyCopyRouter(t, true, 1, test.globalLimit)
			common.TokenKeyReadRateLimitEnable = test.keyLimitEnabled
			common.CriticalRateLimitEnable = test.criticalEnabled
			engine := gin.New()
			require.NoError(t, engine.SetTrustedProxies(nil))
			SetApiRouter(engine)
			user, credential, token := newTokenKeyCopyUser(t)
			ip := fmt.Sprintf("2001:db8:%x::%x", user.Id>>16, user.Id&0xffff)
			path := fmt.Sprintf("/api/token/%d/key", token.Id)
			require.Equal(t, http.StatusOK, requestTokenKeyCopy(engine, path, credential, ip, "").Code)
			assert.Equal(t, test.wantSecond, requestTokenKeyCopy(engine, path, credential, ip, "").Code)
		})
	}
}
