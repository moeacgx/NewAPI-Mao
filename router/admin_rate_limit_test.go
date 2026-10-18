package router

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newManagementRouter(t *testing.T) *gin.Engine {
	t.Helper()
	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	// 限流测试不启动异步审计写入，业务认证及权限中间件保持真实。
	engine.Use(func(c *gin.Context) { common.SetContextKey(c, constant.ContextKeyAuditLogged, true) })
	SetApiRouter(engine)
	return engine
}

func managementRequest(engine http.Handler, method, path, credential, ip, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.RemoteAddr = net.JoinHostPort(ip, "12345")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+credential)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}

func TestAdminManagementRoutesRetainValidationAndPrivileges(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			fixture := setupTokenKeyFixture(t, useRedis, common.RoleRootUser, 1)
			common.GlobalApiRateLimitNum = 1
			fixture.engine = newManagementRouter(t)
			ip := fmt.Sprintf("2001:db8:3::%x:%x", fixture.owner.user.Id>>16, fixture.owner.user.Id&0xffff)
			for range 2 {
				managementRequest(fixture.engine, http.MethodPost, "/api/user/auth/logout", "", ip, "")
			}
			// CT/GA 已耗尽时仍进入管理业务校验，错误请求不得被执行。
			for _, route := range []struct {
				method, path string
				status       int
			}{
				{http.MethodPost, "/api/extensions/conversation-archive/conversations/clear", http.StatusBadRequest},
				{http.MethodPut, "/api/extensions/upstream-model-guard/config", http.StatusBadRequest},
				{http.MethodPut, "/api/extension-admin/test/enabled", http.StatusOK},
				{http.MethodPost, "/api/notification/bots", http.StatusOK},
				{http.MethodDelete, "/api/redemption/batch", http.StatusOK},
				{http.MethodDelete, "/api/promo_code/batch", http.StatusOK},
				{http.MethodDelete, "/api/benefit/admin/activities/batch", http.StatusOK},
				{http.MethodPost, "/api/game/admin/predictions", http.StatusOK},
				{http.MethodPost, "/api/plugin/task/test/dryrun", http.StatusNotImplemented},
			} {
				for _, credential := range []string{fixture.owner.pat, fixture.owner.session} {
					response := managementRequest(fixture.engine, route.method, route.path, credential, ip, "{")
					require.Equal(t, route.status, response.Code, "%s %s: %s", route.method, route.path, response.Body.String())
					var payload struct {
						Success bool `json:"success"`
					}
					require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
					assert.False(t, payload.Success)
				}
			}
			response := managementRequest(fixture.engine, http.MethodPost, "/api/channel/1/key", fixture.owner.session, ip, "")
			assert.Equal(t, http.StatusForbidden, response.Code)
			assert.Contains(t, response.Body.String(), "SECURITY_PROOF_REQUIRED")
			admin := newTokenKeyAccount(t, fixture.db, common.RoleAdminUser)
			for _, path := range []string{"/api/security-audit/config", "/api/security-audit/request-archive/config", "/api/extensions/conversation-archive/config", "/api/extensions/upstream-model-guard/config"} {
				response = managementRequest(fixture.engine, http.MethodGet, path, admin.pat, ip, "")
				assert.Equal(t, http.StatusForbidden, response.Code, path)
				assert.Contains(t, response.Body.String(), "AUTH_INSUFFICIENT_PRIVILEGE")
			}
			response = managementRequest(fixture.engine, http.MethodPost, "/api/plugin/task/test/dryrun", admin.pat, ip, "")
			assert.Equal(t, http.StatusForbidden, response.Code)
		})
	}
}

func TestAdminManagementSearchBypassesUserLimit(t *testing.T) {
	fixture := setupTokenKeyFixture(t, true, common.RoleAdminUser, 1)
	oldEnabled, oldNum, oldDuration := common.SearchRateLimitEnable, common.SearchRateLimitNum, common.SearchRateLimitDuration
	common.SearchRateLimitEnable, common.SearchRateLimitNum, common.SearchRateLimitDuration = true, 1, 60
	t.Cleanup(func() {
		common.SearchRateLimitEnable, common.SearchRateLimitNum, common.SearchRateLimitDuration = oldEnabled, oldNum, oldDuration
	})
	fixture.engine = newManagementRouter(t)
	for _, credential := range []string{fixture.owner.pat, fixture.owner.session, fixture.peer.pat, fixture.peer.session} {
		response := managementRequest(fixture.engine, http.MethodGet, "/api/token/search", credential, "192.0.2.180", "")
		if credential == fixture.peer.session {
			assert.Equal(t, http.StatusTooManyRequests, response.Code)
		} else {
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Contains(t, response.Body.String(), `"success":true`)
		}
	}
}

func TestAdminManagementDoesNotExemptSecurityOrInvalidIdentity(t *testing.T) {
	fixture := setupTokenKeyFixture(t, true, common.RoleRootUser, 1)
	fixture.engine = newManagementRouter(t)
	for index, path := range []string{"/api/user/login", "/api/user/auth/refresh", "/api/verify", "/api/user/auth/logout"} {
		ip := fmt.Sprintf("192.0.2.%d", 181+index)
		managementRequest(fixture.engine, http.MethodPost, path, fixture.owner.session, ip, "{")
		response := managementRequest(fixture.engine, http.MethodPost, path, fixture.owner.session, ip, "{")
		assert.Equal(t, http.StatusTooManyRequests, response.Code, path)
	}
	common.GlobalApiRateLimitNum = 1
	fixture.engine = newManagementRouter(t)
	disabled := newTokenKeyAccount(t, fixture.db, common.RoleAdminUser)
	require.NoError(t, fixture.db.Model(&model.User{}).Where("id = ?", disabled.user.Id).Update("status", common.UserStatusDisabled).Error)
	revoked := newTokenKeyAccount(t, fixture.db, common.RoleAdminUser)
	changed, err := model.RevokeUserSession(revoked.user.Id, fmt.Sprintf("key-session-%d", revoked.user.Id), "test")
	require.NoError(t, err)
	require.True(t, changed)
	for index, credential := range []string{"", "invalid-credential", disabled.pat, disabled.session, revoked.session, fixture.peer.pat, "sk-" + fixture.owner.token.Key} {
		ip := fmt.Sprintf("192.0.2.%d", 200+index)
		managementRequest(fixture.engine, http.MethodGet, "/api/status", "", ip, "")
		request := httptest.NewRequest(http.MethodGet, "/api/token/search", nil)
		request.RemoteAddr = net.JoinHostPort(ip, "12345")
		request.Header.Set("Authorization", "Bearer "+credential)
		request.Header.Set("New-API-User", fmt.Sprint(fixture.owner.user.Id))
		request.Header.Set("role", fmt.Sprint(common.RoleRootUser))
		response := httptest.NewRecorder()
		fixture.engine.ServeHTTP(response, request)
		assert.Equal(t, http.StatusTooManyRequests, response.Code, "credential case %d", index)
	}
}

func TestAdminPersonalSecurityAndFundsRetainGlobalLimit(t *testing.T) {
	for _, role := range []int{common.RoleAdminUser, common.RoleRootUser} {
		t.Run(fmt.Sprintf("role=%d", role), func(t *testing.T) {
			fixture := setupTokenKeyFixture(t, true, role, 1)
			common.GlobalApiRateLimitNum = 1
			fixture.engine = newManagementRouter(t)
			for _, route := range []struct{ method, path string }{
				{http.MethodPost, "/api/user/2fa/enable"},
				{http.MethodPost, "/api/user/passkey/verify/finish"},
				{http.MethodPost, "/api/affiliate/withdraw"},
				{http.MethodPost, "/api/affiliate/transfer-to-balance"},
				{http.MethodPut, "/api/affiliate/payout-account"},
				{http.MethodDelete, "/api/user/self"},
				{http.MethodDelete, "/api/user/sessions/missing-session"},
				{http.MethodPost, "/api/user/sessions/revoke-others"},
			} {
				for _, credentialKind := range []string{"PAT", "Session"} {
					t.Run(route.method+route.path+"/"+credentialKind, func(t *testing.T) {
						// 每个用例使用独立测试账号，防止注销或撤销会话影响后续身份判断。
						account := newTokenKeyAccount(t, fixture.db, role)
						credential := account.pat
						if credentialKind == "Session" {
							credential = account.session
						}
						ip := fmt.Sprintf("2001:db8:4::%x:%x", account.user.Id>>16, account.user.Id&0xffff)
						response := managementRequest(fixture.engine, http.MethodGet, "/api/status", "", ip, "")
						require.Equal(t, http.StatusOK, response.Code)
						response = managementRequest(fixture.engine, route.method, route.path, credential, ip, "{")
						assert.Equal(t, http.StatusTooManyRequests, response.Code)
						assert.NotEmpty(t, response.Header().Get("Retry-After"))
					})
				}
			}
		})
	}
}

func TestAdminManagementBypassesExhaustedGlobalAndTokenKeyLimits(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		for _, role := range []int{common.RoleAdminUser, common.RoleRootUser} {
			t.Run(fmt.Sprintf("redis=%t/role=%d", useRedis, role), func(t *testing.T) {
				fixture := setupTokenKeyFixture(t, useRedis, role, 1)
				common.GlobalApiRateLimitNum = 1
				fixture.engine = gin.New()
				require.NoError(t, fixture.engine.SetTrustedProxies(nil))
				SetApiRouter(fixture.engine)
				ip := fmt.Sprintf("2001:db8:2::%x:%x", fixture.owner.user.Id>>16, fixture.owner.user.Id&0xffff)
				peerPath := fmt.Sprintf("/api/token/%d/key", fixture.peer.token.Id)
				response, _ := fixture.request(t, peerPath, fixture.peer.pat, ip, "")
				require.Equal(t, http.StatusOK, response.Code)
				response, _ = fixture.request(t, peerPath, fixture.peer.pat, ip, "")
				require.Equal(t, http.StatusTooManyRequests, response.Code)
				for _, credential := range []string{fixture.owner.pat, fixture.owner.session} {
					response, payload := fixture.request(t, fmt.Sprintf("/api/token/%d/key", fixture.owner.token.Id), credential, ip, "")
					require.Equal(t, http.StatusOK, response.Code)
					assert.Equal(t, fixture.owner.token.Key, payload.Data.Key)
					response, payload = fixture.request(t, "/api/token/batch/keys", credential, ip, fmt.Sprintf(`{"ids":[%d]}`, fixture.owner.token.Id))
					require.Equal(t, http.StatusOK, response.Code)
					assert.True(t, payload.Success)
				}
			})
		}
	}
}
