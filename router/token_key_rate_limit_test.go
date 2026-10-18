package router

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var tokenKeyUserSequence atomic.Int64

type tokenKeyAccount struct {
	user    model.User
	token   model.Token
	pat     string
	session string
}

type tokenKeyFixture struct {
	engine *gin.Engine
	db     *gorm.DB
	redis  *miniredis.Miniredis
	owner  tokenKeyAccount
	peer   tokenKeyAccount
}

type tokenKeyResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Key  string            `json:"key"`
		Keys map[string]string `json:"keys"`
	} `json:"data"`
}

func newTokenKeyAccount(t *testing.T, db *gorm.DB, role int) tokenKeyAccount {
	t.Helper()
	id := int(tokenKeyUserSequence.Add(1)) + 10000000
	pat := fmt.Sprintf("token-key-pat-%d", id)
	account := tokenKeyAccount{pat: pat}
	account.user = model.User{Id: id, Username: fmt.Sprintf("key-user-%d", id), Role: role,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1, AccessToken: &pat,
		AffCode: fmt.Sprintf("key-aff-%d", id)}
	require.NoError(t, db.Create(&account.user).Error)
	account.token = model.Token{UserId: id, Name: "fixture-key", Key: fmt.Sprintf("keyfixture%d", id),
		Status: common.TokenStatusEnabled, UnlimitedQuota: true, ExpiredTime: -1}
	require.NoError(t, db.Create(&account.token).Error)
	session := model.UserSession{SID: fmt.Sprintf("key-session-%d", id), UserID: id, Version: 1,
		UserAuthVersion: 1, Status: model.UserSessionStatusActive, ExpiresAt: time.Now().Unix() + 3600,
		RefreshHash: fmt.Sprintf("fixture-refresh-%d", id), LoginMethod: "password"}
	require.NoError(t, model.CreateUserSession(&session))
	var err error
	account.session, _, err = service.IssueAccessToken(service.AuthIdentity{
		UserID: id, SessionID: session.SID, UserAuthVersion: 1, SessionVersion: 1,
	})
	require.NoError(t, err)
	return account
}

func setupTokenKeyFixture(t *testing.T, useRedis bool, role, keyLimit int) tokenKeyFixture {
	t.Helper()
	oldDB, oldRedis, oldClient := model.DB, common.RedisEnabled, common.RDB
	oldType, oldSecret := common.MainDatabaseType(), common.SessionSecret
	oldGA, oldGANum, oldGADuration := common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration
	oldCT, oldCTNum, oldCTDuration := common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration
	oldKey, oldKeyNum, oldKeyDuration := common.TokenKeyReadRateLimitEnable, common.TokenKeyReadRateLimitNum, common.TokenKeyReadRateLimitDuration
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	model.InitDBColumns()
	common.SessionSecret = "token-key-fixture-session-secret"
	common.RedisEnabled = useRedis
	common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration = true, 1000, 60
	common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = true, 1, 60
	common.TokenKeyReadRateLimitEnable, common.TokenKeyReadRateLimitNum, common.TokenKeyReadRateLimitDuration = true, keyLimit, 60
	t.Cleanup(func() {
		model.DB, common.RedisEnabled, common.RDB = oldDB, oldRedis, oldClient
		common.SetMainDatabaseType(oldType)
		model.InitDBColumns()
		common.SessionSecret = oldSecret
		common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration = oldGA, oldGANum, oldGADuration
		common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = oldCT, oldCTNum, oldCTDuration
		common.TokenKeyReadRateLimitEnable, common.TokenKeyReadRateLimitNum, common.TokenKeyReadRateLimitDuration = oldKey, oldKeyNum, oldKeyDuration
	})
	fixture := tokenKeyFixture{}
	if useRedis {
		fixture.redis = miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: fixture.redis.Addr()})
		common.RDB = client
		t.Cleanup(func() { require.NoError(t, client.Close()) })
	}
	var err error
	fixture.db, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "keys.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := fixture.db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	model.DB = fixture.db
	require.NoError(t, fixture.db.AutoMigrate(&model.User{}, &model.Token{}, &model.UserSession{}, &model.Group{}))
	require.NoError(t, fixture.db.Create(&model.Group{Code: "default", Name: "测试分组", Status: model.GroupStatusActive}).Error)
	fixture.owner = newTokenKeyAccount(t, fixture.db, role)
	fixture.peer = newTokenKeyAccount(t, fixture.db, common.RoleCommonUser)
	gin.SetMode(gin.TestMode)
	fixture.engine = gin.New()
	require.NoError(t, fixture.engine.SetTrustedProxies(nil))
	SetApiRouter(fixture.engine)
	return fixture
}

func (fixture tokenKeyFixture) request(t *testing.T, path, credential, ip, body string) (*httptest.ResponseRecorder, tokenKeyResponse) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.RemoteAddr = net.JoinHostPort(ip, "12345")
	request.Header.Set("Content-Type", "application/json")
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response := httptest.NewRecorder()
	fixture.engine.ServeHTTP(response, request)
	var payload tokenKeyResponse
	if response.Body.Len() > 0 {
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	}
	return response, payload
}

func TestTokenKeyReadIsolatedFromCriticalIPBucket(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		for _, role := range []int{common.RoleCommonUser, common.RoleAdminUser, common.RoleRootUser} {
			t.Run(fmt.Sprintf("redis=%t/role=%d", useRedis, role), func(t *testing.T) {
				fixture := setupTokenKeyFixture(t, useRedis, role, 3)
				ip := fmt.Sprintf("2001:db8::%x:%x", fixture.owner.user.Id>>16, fixture.owner.user.Id&0xffff)
				response, _ := fixture.request(t, "/api/user/auth/logout", "", ip, "")
				require.Equal(t, http.StatusOK, response.Code)
				response, _ = fixture.request(t, "/api/user/auth/logout", "", ip, "")
				require.Equal(t, http.StatusTooManyRequests, response.Code)
				path := fmt.Sprintf("/api/token/%d/key", fixture.owner.token.Id)
				response, payload := fixture.request(t, path, fixture.owner.pat, ip, "")
				require.Equal(t, http.StatusOK, response.Code)
				require.True(t, payload.Success)
				assert.Equal(t, fixture.owner.token.Key, payload.Data.Key)
				response, payload = fixture.request(t, "/api/token/batch/keys", fixture.owner.session, ip, fmt.Sprintf(`{"ids":[%d]}`, fixture.owner.token.Id))
				require.Equal(t, http.StatusOK, response.Code)
				require.True(t, payload.Success)
				assert.Equal(t, map[string]string{strconv.Itoa(fixture.owner.token.Id): fixture.owner.token.Key}, payload.Data.Keys)
				otherIP := fmt.Sprintf("2001:db8:1::%x:%x", fixture.owner.user.Id>>16, fixture.owner.user.Id&0xffff)
				response, _ = fixture.request(t, path, fixture.owner.pat, otherIP, "")
				require.Equal(t, http.StatusOK, response.Code)
				response, _ = fixture.request(t, "/api/user/auth/logout", "", otherIP, "")
				assert.Equal(t, http.StatusOK, response.Code)
			})
		}
	}
}

func TestTokenKeyReadSharesUserQuotaAcrossRoutesCredentialsAndIPs(t *testing.T) {
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			fixture := setupTokenKeyFixture(t, useRedis, common.RoleCommonUser, 2)
			path := fmt.Sprintf("/api/token/%d/key", fixture.owner.token.Id)
			batch := fmt.Sprintf(`{"ids":[%d]}`, fixture.owner.token.Id)
			response, payload := fixture.request(t, path, fixture.owner.pat, "192.0.2.150", "")
			require.Equal(t, http.StatusOK, response.Code)
			require.True(t, payload.Success)
			assert.Contains(t, response.Header().Get("Cache-Control"), "no-store")
			response, payload = fixture.request(t, "/api/token/batch/keys", fixture.owner.session, "192.0.2.151", batch)
			require.Equal(t, http.StatusOK, response.Code)
			require.True(t, payload.Success)
			// 两个入口、不同凭证和不同 IP 仍消耗同一个用户额度。
			for _, request := range []struct{ path, body string }{{path, ""}, {"/api/token/batch/keys", batch}} {
				response, _ = fixture.request(t, request.path, fixture.owner.pat, "192.0.2.152", request.body)
				assert.Equal(t, http.StatusTooManyRequests, response.Code)
				assert.Equal(t, "60", response.Header().Get("Retry-After"))
				assert.Contains(t, response.Header().Get("Cache-Control"), "no-store")
				assert.Empty(t, response.Body.String())
			}
			// 共享出口的其他用户仍能获取自己的密钥。
			peerPath := fmt.Sprintf("/api/token/%d/key", fixture.peer.token.Id)
			response, payload = fixture.request(t, peerPath, fixture.peer.pat, "192.0.2.150", "")
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, fixture.peer.token.Key, payload.Data.Key)
			if fixture.redis != nil {
				fixture.redis.FastForward(60 * time.Second)
				response, payload = fixture.request(t, path, fixture.owner.pat, "192.0.2.150", "")
				assert.Equal(t, http.StatusOK, response.Code)
				assert.Equal(t, fixture.owner.token.Key, payload.Data.Key)
			}
		})
	}
}

func TestTokenKeyReadPreservesOwnershipForEveryRole(t *testing.T) {
	for _, role := range []int{common.RoleCommonUser, common.RoleAdminUser, common.RoleRootUser} {
		t.Run(fmt.Sprint(role), func(t *testing.T) {
			fixture := setupTokenKeyFixture(t, false, role, 3)
			path := fmt.Sprintf("/api/token/%d/key", fixture.peer.token.Id)
			response, payload := fixture.request(t, path, fixture.owner.pat, "192.0.2.153", "")
			assert.False(t, payload.Success)
			assert.NotContains(t, response.Body.String(), fixture.peer.token.Key)
			response, payload = fixture.request(t, "/api/token/batch/keys", fixture.owner.session, "192.0.2.153",
				fmt.Sprintf(`{"ids":[%d,%d]}`, fixture.owner.token.Id, fixture.peer.token.Id))
			require.Equal(t, http.StatusOK, response.Code)
			require.True(t, payload.Success)
			assert.Equal(t, map[string]string{strconv.Itoa(fixture.owner.token.Id): fixture.owner.token.Key}, payload.Data.Keys)
		})
	}
}

func TestTokenKeyReadRejectsUnauthenticatedAndDisabledUsers(t *testing.T) {
	fixture := setupTokenKeyFixture(t, true, common.RoleAdminUser, 1)
	disabled := newTokenKeyAccount(t, fixture.db, common.RoleAdminUser)
	require.NoError(t, fixture.db.Model(&model.User{}).Where("id = ?", disabled.user.Id).Update("status", common.UserStatusDisabled).Error)
	for _, credential := range []string{"", "invalid-credential", disabled.pat, disabled.session} {
		for _, path := range []string{fmt.Sprintf("/api/token/%d/key", fixture.owner.token.Id), "/api/token/batch/keys"} {
			response, _ := fixture.request(t, path, credential, "192.0.2.154", fmt.Sprintf(`{"ids":[%d]}`, fixture.owner.token.Id))
			assert.Equal(t, http.StatusUnauthorized, response.Code)
			assert.NotContains(t, response.Body.String(), fixture.owner.token.Key)
		}
	}
	response, payload := fixture.request(t, fmt.Sprintf("/api/token/%d/key", fixture.owner.token.Id), fixture.owner.pat, "192.0.2.154", "")
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, fixture.owner.token.Key, payload.Data.Key)
}

func TestTokenKeyReadBatchBoundAndOneRequestAccounting(t *testing.T) {
	fixture := setupTokenKeyFixture(t, true, common.RoleCommonUser, 2)
	tokens := make([]model.Token, 100)
	for index := range tokens {
		tokens[index] = model.Token{UserId: fixture.owner.user.Id, Key: fmt.Sprintf("batchkey%d", index)}
	}
	require.NoError(t, fixture.db.Create(&tokens).Error)
	ids := make([]int, len(tokens))
	expected := make(map[string]string, len(tokens))
	for index, token := range tokens {
		ids[index] = token.Id
		expected[strconv.Itoa(token.Id)] = token.Key
	}
	body, err := common.Marshal(map[string]any{"ids": ids})
	require.NoError(t, err)
	response, payload := fixture.request(t, "/api/token/batch/keys", fixture.owner.pat, "192.0.2.155", string(body))
	require.Equal(t, http.StatusOK, response.Code)
	require.True(t, payload.Success)
	assert.Equal(t, expected, payload.Data.Keys)
	// 一次批量不会按返回密钥数消耗请求额度。
	response, payload = fixture.request(t, fmt.Sprintf("/api/token/%d/key", fixture.owner.token.Id), fixture.owner.pat, "192.0.2.155", "")
	require.Equal(t, http.StatusOK, response.Code)
	require.True(t, payload.Success)
	fixture.redis.FastForward(time.Minute)
	body, err = common.Marshal(map[string]any{"ids": append(ids, fixture.owner.token.Id)})
	require.NoError(t, err)
	response, payload = fixture.request(t, "/api/token/batch/keys", fixture.owner.pat, "192.0.2.155", string(body))
	assert.Equal(t, http.StatusOK, response.Code)
	assert.False(t, payload.Success)
	assert.Empty(t, payload.Data.Keys)
}

func TestTokenKeyReadLimiterSwitchRetainsGAAndAuthentication(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			fixture := setupTokenKeyFixture(t, true, common.RoleCommonUser, 3)
			common.TokenKeyReadRateLimitEnable = enabled
			common.GlobalApiRateLimitNum = 2
			fixture.engine = gin.New()
			require.NoError(t, fixture.engine.SetTrustedProxies(nil))
			SetApiRouter(fixture.engine)
			path := fmt.Sprintf("/api/token/%d/key", fixture.owner.token.Id)
			for range 2 {
				response, payload := fixture.request(t, path, fixture.owner.pat, "192.0.2.156", "")
				require.Equal(t, http.StatusOK, response.Code)
				require.True(t, payload.Success)
			}
			response, _ := fixture.request(t, path, fixture.owner.pat, "192.0.2.156", "")
			assert.Equal(t, http.StatusTooManyRequests, response.Code)
			response, _ = fixture.request(t, path, "", "192.0.2.157", "")
			assert.Equal(t, http.StatusUnauthorized, response.Code)
			response, payload := fixture.request(t, fmt.Sprintf("/api/token/%d/key", fixture.peer.token.Id), fixture.owner.pat, "192.0.2.158", "")
			assert.False(t, payload.Success)
			assert.NotContains(t, response.Body.String(), fixture.peer.token.Key)
			if !enabled {
				// 关闭专用额度后累计请求可超过原额度，GA 仍按各来源保护。
				response, payload = fixture.request(t, path, fixture.owner.pat, "192.0.2.159", "")
				assert.Equal(t, http.StatusOK, response.Code)
				assert.True(t, payload.Success)
			}
		})
	}
}

func TestTokenKeyReadLimitIndependentOfCriticalSwitch(t *testing.T) {
	fixture := setupTokenKeyFixture(t, true, common.RoleCommonUser, 1)
	common.CriticalRateLimitEnable = false
	fixture.engine = gin.New()
	require.NoError(t, fixture.engine.SetTrustedProxies(nil))
	SetApiRouter(fixture.engine)
	path := fmt.Sprintf("/api/token/%d/key", fixture.owner.token.Id)
	response, payload := fixture.request(t, path, fixture.owner.pat, "192.0.2.160", "")
	require.Equal(t, http.StatusOK, response.Code)
	require.True(t, payload.Success)
	response, _ = fixture.request(t, path, fixture.owner.pat, "192.0.2.160", "")
	assert.Equal(t, http.StatusTooManyRequests, response.Code)
}
