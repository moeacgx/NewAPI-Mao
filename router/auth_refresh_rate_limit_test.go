package router

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var authRefreshClientSequence atomic.Uint64

func setupAuthRefreshRateLimitRouter(t *testing.T, useRedis bool, globalLimit int) (*gin.Engine, *miniredis.Miniredis) {
	t.Helper()
	previousRedis, previousClient := common.RedisEnabled, common.RDB
	previousGlobal, previousGlobalNum, previousGlobalDuration := common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration
	previousCritical, previousCriticalNum, previousCriticalDuration := common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration
	common.RedisEnabled = useRedis
	common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration = true, globalLimit, 30
	common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = true, 1, 30
	var redisServer *miniredis.Miniredis
	if useRedis {
		redisServer = miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
		common.RDB = client
		t.Cleanup(func() { require.NoError(t, client.Close()) })
	}
	t.Cleanup(func() {
		common.RedisEnabled, common.RDB = previousRedis, previousClient
		common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration = previousGlobal, previousGlobalNum, previousGlobalDuration
		common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = previousCritical, previousCriticalNum, previousCriticalDuration
	})
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	SetApiRouter(engine)
	return engine, redisServer
}

func requestAuthRefreshRateLimit(engine *gin.Engine, path, clientIP string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, nil)
	request.RemoteAddr = net.JoinHostPort(clientIP, "12345")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}

func TestAuthRefreshIsolatedFromCriticalRequests(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		useRedis bool
	}{
		{name: "memory"},
		{name: "redis", useRedis: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			engine, redisServer := setupAuthRefreshRateLimitRouter(t, testCase.useRedis, 100)
			// 进程内限流器跨路由存活，每次夹具分配独立来源以支持重复运行。
			clientIP := fmt.Sprintf("2001:db8::%x", authRefreshClientSequence.Add(1))
			assert.Equal(t, http.StatusOK, requestAuthRefreshRateLimit(engine, "/api/user/auth/logout", clientIP).Code)
			assert.Equal(t, http.StatusTooManyRequests, requestAuthRefreshRateLimit(engine, "/api/user/auth/logout", clientIP).Code)

			// 没有 Cookie 应由刷新控制器返回 401，不能被其他接口耗尽的 CT 提前拦截。
			response := requestAuthRefreshRateLimit(engine, "/api/user/auth/refresh", clientIP)
			require.Equal(t, http.StatusUnauthorized, response.Code)
			response = requestAuthRefreshRateLimit(engine, "/api/user/auth/refresh", clientIP)
			assert.Equal(t, http.StatusTooManyRequests, response.Code)
			assert.Equal(t, "30", response.Header().Get("Retry-After"))
			assert.Empty(t, response.Header().Values("Set-Cookie"))

			// 反向验证：刷新也不应占用登录、注销等关键接口的 CT 配额。
			otherIP := fmt.Sprintf("2001:db8::%x", authRefreshClientSequence.Add(1))
			assert.Equal(t, http.StatusUnauthorized, requestAuthRefreshRateLimit(engine, "/api/user/auth/refresh", otherIP).Code)
			assert.Equal(t, http.StatusOK, requestAuthRefreshRateLimit(engine, "/api/user/auth/logout", otherIP).Code)
			if redisServer != nil {
				redisServer.FastForward(30 * time.Second)
				assert.Equal(t, http.StatusUnauthorized, requestAuthRefreshRateLimit(engine, "/api/user/auth/refresh", clientIP).Code)
			}
		})
	}
}

func TestAuthRefreshStillUsesGlobalAPIRateLimit(t *testing.T) {
	engine, _ := setupAuthRefreshRateLimitRouter(t, true, 1)
	const clientIP = "192.0.2.142"
	assert.Equal(t, http.StatusOK, requestAuthRefreshRateLimit(engine, "/api/user/auth/logout", clientIP).Code)
	response := requestAuthRefreshRateLimit(engine, "/api/user/auth/refresh", clientIP)
	assert.Equal(t, http.StatusTooManyRequests, response.Code)
	assert.Equal(t, "30", response.Header().Get("Retry-After"))
	assert.Empty(t, response.Header().Values("Set-Cookie"))
}
