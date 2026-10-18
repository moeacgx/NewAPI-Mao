package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAuthCompatibilityDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.AuthFlow{}, &model.ExternalIdentityClaim{}, &model.Log{}))
	oldDB, oldLogDB, oldRedis, oldType := model.DB, model.LOG_DB, common.RedisEnabled, common.MainDatabaseType()
	model.DB, model.LOG_DB, common.RedisEnabled = db, db, false
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		model.DB, model.LOG_DB, common.RedisEnabled = oldDB, oldLogDB, oldRedis
		common.SetMainDatabaseType(oldType)
		_ = sqlDB.Close()
	})
	return db
}

func TestSetupLoginRejectsStaleCredentialVersion(t *testing.T) {
	db := setupAuthCompatibilityDB(t)
	user := &model.User{Username: "stale-login", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, AuthVersion: 1}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Model(user).Update("auth_version", 2).Error)
	user.AuthVersion = 1
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/login", nil)
	setupLogin(user, c)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, w.Header().Values("Set-Cookie"))
	var count int64
	require.NoError(t, db.Model(&model.UserSession{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestTelegramBindRejectsAdvancedSessionVersion(t *testing.T) {
	for _, change := range []string{"session_version", "user_and_session_version"} {
		t.Run(change, func(t *testing.T) {
			db := setupAuthCompatibilityDB(t)
			oldEnabled, oldToken := common.TelegramOAuthEnabled, common.TelegramBotToken
			common.TelegramOAuthEnabled, common.TelegramBotToken = true, "compat-test-bot"
			t.Cleanup(func() { common.TelegramOAuthEnabled, common.TelegramBotToken = oldEnabled, oldToken })
			user, token := createTelegramBindTestFlow(t, db, "version-test", common.UserStatusEnabled, time.Now())
			// 固定旧版本 payload，模拟绑定发起后同一个 SID 被推进。
			payload, err := common.Marshal(service.AuthIdentity{UserID: user.Id, SessionID: "version-test-session", UserAuthVersion: 1, SessionVersion: 1})
			require.NoError(t, err)
			require.NoError(t, db.Model(&model.AuthFlow{}).Where("purpose = ?", model.AuthFlowPurposeTelegramBind).Update("payload", string(payload)).Error)
			updates := map[string]any{"version": 2}
			if change == "user_and_session_version" {
				require.NoError(t, db.Model(user).Update("auth_version", 2).Error)
				updates["user_auth_version"] = 2
			}
			require.NoError(t, db.Model(&model.UserSession{}).Where("user_id = ?", user.Id).Updates(updates).Error)
			params := signedTelegramAuthorization(common.TelegramBotToken, time.Now())
			r := gin.New()
			r.GET("/api/oauth/telegram/bind/:flow_token", TelegramBind)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/oauth/telegram/bind/"+token+"?"+params.Encode(), nil))
			assertTelegramBindRedirect(t, w, token, telegramBindErrorSessionInvalid)
			require.NoError(t, db.First(user, user.Id).Error)
			assert.Empty(t, user.TelegramId)
			_, err = model.GetAuthFlow(token, model.AuthFlowMatch{Purpose: model.AuthFlowPurposeTelegramBind})
			assert.NoError(t, err, "失败不得消费绑定流程")
		})
	}
}

func TestTelegramBindStartFreezesIdentityAndRejectsUnversionedFlow(t *testing.T) {
	db := setupAuthCompatibilityDB(t)
	oldEnabled, oldToken := common.TelegramOAuthEnabled, common.TelegramBotToken
	common.TelegramOAuthEnabled, common.TelegramBotToken = true, "compat-start-bot"
	t.Cleanup(func() { common.TelegramOAuthEnabled, common.TelegramBotToken = oldEnabled, oldToken })
	user, _ := createTelegramBindTestFlow(t, db, "start-version", common.UserStatusEnabled, time.Now())
	identity := service.AuthIdentity{UserID: user.Id, SessionID: "start-version-session", UserAuthVersion: 1, SessionVersion: 1}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/oauth/telegram/bind/start", nil)
	c.Set("auth_identity", identity)
	TelegramBindStart(c)
	var response struct {
		Success bool
		Data    struct {
			FlowToken string `json:"flow_token"`
		}
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	flow, err := model.GetAuthFlow(response.Data.FlowToken, model.AuthFlowMatch{Purpose: model.AuthFlowPurposeTelegramBind})
	require.NoError(t, err)
	var storedIdentity service.AuthIdentity
	require.NoError(t, common.UnmarshalJsonStr(flow.Payload, &storedIdentity))
	assert.Equal(t, identity, storedIdentity)
	// 升级前的空 payload 必须失败关闭，不从当前会话补造历史版本。
	require.NoError(t, db.Model(flow).Update("payload", "").Error)
	r := gin.New()
	r.GET("/api/oauth/telegram/bind/:flow_token", TelegramBind)
	w = httptest.NewRecorder()
	params := signedTelegramAuthorization(common.TelegramBotToken, time.Now())
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/oauth/telegram/bind/"+response.Data.FlowToken+"?"+params.Encode(), nil))
	assertTelegramBindRedirect(t, w, response.Data.FlowToken, telegramBindErrorFlowInvalid)
}

func TestGenerateAccessTokenRecordsRedactedAudit(t *testing.T) {
	db := setupAuthCompatibilityDB(t)
	user := &model.User{Username: "pat-audit", Status: common.UserStatusEnabled, AuthVersion: 1}
	require.NoError(t, db.Create(user).Error)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/token", nil)
	c.Set("id", user.Id)
	GenerateAccessToken(c)
	var response struct {
		Success bool
		Data    string
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	var logs []model.Log
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Contains(t, logs[0].Other, "access_token.generate")
	assert.NotContains(t, logs[0].Other, response.Data)
	assert.NotContains(t, logs[0].Content, response.Data)
}

func TestRevokeAccessTokenIsIdempotentAndPreservesSession(t *testing.T) {
	db := setupAuthCompatibilityDB(t)
	user := &model.User{Username: "pat-revoke", Status: common.UserStatusEnabled, AuthVersion: 1}
	user.SetAccessToken("old-personal-access-token")
	require.NoError(t, db.Create(user).Error)
	bundle, err := service.CreateLoginSession(user.Id, "password", "127.0.0.1", "test")
	require.NoError(t, err)
	identity, err := service.ParseAccessToken(bundle.AccessToken)
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/api/user/token", nil)
		c.Set("id", user.Id)
		RevokeAccessToken(c)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"success":true`)
	}
	var stored model.User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Nil(t, stored.AccessToken)
	assert.Equal(t, int64(1), stored.AuthVersion)
	_, _, err = service.ValidateLoginSession(identity)
	assert.NoError(t, err)
	var logs []model.Log
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Contains(t, logs[0].Other, "access_token.revoke")
	assert.Contains(t, logs[0].Other, model.AccessTokenFingerprint("old-personal-access-token"))
	assert.NotContains(t, logs[0].Other, "old-personal-access-token")
}

func TestPasskeyEnrollmentRejectsRevokedSessionAtWrite(t *testing.T) {
	db := setupAuthCompatibilityDB(t)
	require.NoError(t, db.AutoMigrate(&model.PasskeyCredential{}))
	user := &model.User{Username: "passkey-write", Status: common.UserStatusEnabled, AuthVersion: 1}
	require.NoError(t, db.Create(user).Error)
	bundle, err := service.CreateLoginSession(user.Id, "password", "127.0.0.1", "test")
	require.NoError(t, err)
	identity, err := service.ParseAccessToken(bundle.AccessToken)
	require.NoError(t, err)
	_, err = model.RevokeUserSession(user.Id, bundle.Session.SID, "test")
	require.NoError(t, err)
	credential := &model.PasskeyCredential{UserID: user.Id, CredentialID: "credential", PublicKey: "key"}
	err = model.RegisterPasskeyForSession(identity, credential)
	assert.Error(t, err, "完成 WebAuthn 验证后撤销会话，最终凭据写入必须拒绝")
	var count int64
	require.NoError(t, db.Model(&model.PasskeyCredential{}).Count(&count).Error)
	assert.Zero(t, count)
}
