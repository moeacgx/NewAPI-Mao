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
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var tokenQueryUserSequence atomic.Int64

func setupTokenQueryRouter(t *testing.T, useRedis bool, queryLimit, globalLimit int) (*gin.Engine, *miniredis.Miniredis) {
	t.Helper()
	t.Cleanup(model.InitDBColumns)
	db := setupFeatureRouterAuthTest(t)
	model.InitDBColumns()
	require.NoError(t, db.AutoMigrate(&model.Token{}, &model.TokenGroupBinding{}, &model.Log{}))
	previousLogDB, previousLogType := model.LOG_DB, common.LogDatabaseType()
	model.LOG_DB = db
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	previousEnable, previousNum, previousDuration := common.TokenQueryRateLimitEnable, common.TokenQueryRateLimitNum, common.TokenQueryRateLimitDuration
	common.TokenQueryRateLimitEnable, common.TokenQueryRateLimitNum, common.TokenQueryRateLimitDuration = true, queryLimit, 30
	t.Cleanup(func() {
		model.LOG_DB = previousLogDB
		common.SetLogDatabaseType(previousLogType)
		common.TokenQueryRateLimitEnable, common.TokenQueryRateLimitNum, common.TokenQueryRateLimitDuration = previousEnable, previousNum, previousDuration
	})
	return setupAuthRefreshRateLimitRouter(t, useRedis, globalLimit)
}

func newTokenQueryUser(t *testing.T, status int) (*model.User, string, *model.Token) {
	t.Helper()
	id := int(tokenQueryUserSequence.Add(1)) + 20000000
	credential := fmt.Sprintf("query-test-pat-%d", id)
	user := &model.User{Id: id, Username: fmt.Sprintf("query-user-%d", id), Role: common.RoleCommonUser,
		Status: status, Group: "default", AccessToken: &credential,
		AffCode: fmt.Sprintf("query-aff-%d", id), AuthVersion: 1}
	require.NoError(t, model.DB.Create(user).Error)
	token := &model.Token{UserId: id, Key: fmt.Sprintf("query%032d", id), Name: "query-token",
		GroupMode: model.TokenGroupModeInherit, Status: common.TokenStatusEnabled, ExpiredTime: -1, RemainQuota: 42, UsedQuota: 8}
	require.NoError(t, model.DB.Create(token).Error)
	return user, credential, token
}

func requestTokenQuery(engine *gin.Engine, path, key, ip string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr = net.JoinHostPort(ip, "12345")
	if key != "" {
		request.Header.Set("Authorization", "Bearer sk-"+key)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}

func TestTokenQueriesAndPaymentCriticalQuotaAreIndependent(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			engine, redisServer := setupTokenQueryRouter(t, useRedis, 1, 100)
			user, credential, token := newTokenQueryUser(t, common.UserStatusEnabled)
			ip := fmt.Sprintf("2001:db8:%x::%x", user.Id>>16, user.Id&0xffff)
			for _, path := range []string{"/api/usage/token/", "/api/log/token"} {
				response := requestTokenQuery(engine, path, token.Key, ip)
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				if strings.Contains(path, "usage") {
					assert.Contains(t, response.Body.String(), `"total_available":42`)
				} else {
					assert.Contains(t, response.Body.String(), `"success":true`)
				}
				blocked := requestTokenQuery(engine, path, token.Key, ip)
				assert.Equal(t, http.StatusTooManyRequests, blocked.Code)
				assert.Equal(t, "30", blocked.Header().Get("Retry-After"))
			}

			// 无效请求体使真实支付控制器在调用支付商前返回，验证查询限流不会阻塞充值。
			payment := requestTokenKeyCopy(engine, "/api/user/pay", credential, ip, "{")
			require.Equal(t, http.StatusOK, payment.Code, payment.Body.String())
			assert.Contains(t, payment.Body.String(), "参数错误")
			assert.Equal(t, http.StatusTooManyRequests, requestTokenKeyCopy(engine, "/api/user/pay", credential, ip, "{").Code)

			// 另一个出口先耗尽充值 CT，首次余额和日志查询仍须成功。
			otherUser, otherCredential, otherToken := newTokenQueryUser(t, common.UserStatusEnabled)
			otherIP := fmt.Sprintf("2001:db8:%x::%x", otherUser.Id>>16, otherUser.Id&0xffff)
			require.Equal(t, http.StatusOK, requestTokenKeyCopy(engine, "/api/user/pay", otherCredential, otherIP, "{").Code)
			require.Equal(t, http.StatusTooManyRequests, requestTokenKeyCopy(engine, "/api/user/pay", otherCredential, otherIP, "{").Code)
			for _, path := range []string{"/api/usage/token/", "/api/log/token"} {
				assert.Equal(t, http.StatusOK, requestTokenQuery(engine, path, otherToken.Key, otherIP).Code)
			}
			if redisServer != nil {
				redisServer.FastForward(30 * time.Second)
				assert.Equal(t, http.StatusOK, requestTokenQuery(engine, "/api/usage/token/", token.Key, ip).Code)
			}
		})
	}
}

func TestTokenQueryQuotaTracksOwnerAcrossKeysAndIPs(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			engine, _ := setupTokenQueryRouter(t, useRedis, 1, 100)
			user, _, token := newTokenQueryUser(t, common.UserStatusEnabled)
			_, _, otherToken := newTokenQueryUser(t, common.UserStatusEnabled)
			secondToken := &model.Token{UserId: user.Id, Key: fmt.Sprintf("alternate%032d", user.Id),
				Status: common.TokenStatusEnabled, GroupMode: model.TokenGroupModeInherit, ExpiredTime: -1}
			require.NoError(t, model.DB.Create(secondToken).Error)
			ip := fmt.Sprintf("2001:db8:%x::%x", user.Id>>16, user.Id&0xffff)
			otherIP := fmt.Sprintf("2001:db8:%x:1::%x", user.Id>>16, user.Id&0xffff)
			for _, path := range []string{"/api/usage/token/", "/api/log/token"} {
				require.Equal(t, http.StatusOK, requestTokenQuery(engine, path, token.Key, ip).Code)
				assert.Equal(t, http.StatusTooManyRequests, requestTokenQuery(engine, path, secondToken.Key, otherIP).Code,
					"同用户换密钥和出口不能绕过查询额度")
				assert.Equal(t, http.StatusOK, requestTokenQuery(engine, path, otherToken.Key, ip).Code,
					"同出口不同用户的查询额度隔离")
			}
		})
	}
}

func TestTokenQueryAuthenticationStillPrecedesQueryLimit(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			engine, _ := setupTokenQueryRouter(t, useRedis, 1, 100)
			user, _, token := newTokenQueryUser(t, common.UserStatusEnabled)
			_, _, bannedToken := newTokenQueryUser(t, common.UserStatusDisabled)
			disabled := &model.Token{UserId: user.Id, Key: fmt.Sprintf("disabled%032d", user.Id),
				Status: common.TokenStatusDisabled, GroupMode: model.TokenGroupModeInherit}
			require.NoError(t, model.DB.Create(disabled).Error)
			// 保留只读查询允许已过期、额度耗尽令牌的现有契约。
			require.NoError(t, model.DB.Model(token).Updates(map[string]interface{}{
				"status": common.TokenStatusExhausted, "expired_time": int64(1), "remain_quota": 0,
			}).Error)
			ip := fmt.Sprintf("2001:db8:%x::%x", user.Id>>16, user.Id&0xffff)
			for _, path := range []string{"/api/usage/token/", "/api/log/token"} {
				for _, invalidKey := range []string{"", "unknown", disabled.Key} {
					assert.Equal(t, http.StatusUnauthorized, requestTokenQuery(engine, path, invalidKey, ip).Code)
				}
				assert.Equal(t, http.StatusForbidden, requestTokenQuery(engine, path, bannedToken.Key, ip).Code)
				assert.Equal(t, http.StatusOK, requestTokenQuery(engine, path, token.Key, ip).Code,
					"失败认证不得耗尽所有者查询额度")
				assert.Equal(t, http.StatusTooManyRequests, requestTokenQuery(engine, path, token.Key, ip).Code)
				assert.Equal(t, http.StatusUnauthorized, requestTokenQuery(engine, path, disabled.Key, ip).Code)
			}
		})
	}
}

func TestTokenQueryLimitSwitchDoesNotDisableGlobalProtection(t *testing.T) {
	for _, testCase := range []struct {
		name                    string
		queryEnabled, ctEnabled bool
		globalLimit, wantSecond int
	}{
		{"关闭查询限流不关闭GA", false, true, 1, http.StatusTooManyRequests},
		{"关闭CT不关闭查询限流", true, false, 100, http.StatusTooManyRequests},
		{"查询限流独立开关", false, true, 100, http.StatusOK},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setupTokenQueryRouter(t, true, 1, testCase.globalLimit)
			common.TokenQueryRateLimitEnable, common.CriticalRateLimitEnable = testCase.queryEnabled, testCase.ctEnabled
			engine := gin.New()
			require.NoError(t, engine.SetTrustedProxies(nil))
			SetApiRouter(engine)
			user, _, token := newTokenQueryUser(t, common.UserStatusEnabled)
			ip := fmt.Sprintf("2001:db8:%x::%x", user.Id>>16, user.Id&0xffff)
			require.Equal(t, http.StatusOK, requestTokenQuery(engine, "/api/usage/token/", token.Key, ip).Code)
			assert.Equal(t, testCase.wantSecond, requestTokenQuery(engine, "/api/usage/token/", token.Key, ip).Code)
		})
	}
}
