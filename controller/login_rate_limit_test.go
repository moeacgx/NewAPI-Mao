package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	appI18n "github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPasswordLoginRateLimitDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.AuthFlow{}, &model.TwoFA{}, &model.Log{}))

	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedis := common.RedisEnabled
	previousDatabaseType := common.MainDatabaseType()
	previousPasswordLoginEnabled := common.PasswordLoginEnabled
	previousRateLimitEnabled := common.LoginFailureRateLimitEnable
	previousAccountLimit := common.LoginFailureRateLimitNum
	previousIPLimit := common.LoginFailureIPRateLimitNum
	previousDuration := common.LoginFailureRateLimitDuration
	previousInflightLimit := common.LoginInflightIPLimit
	previousLeaseDuration := common.LoginInflightLeaseDuration
	previousSessionSecret := common.SessionSecret
	model.DB, model.LOG_DB = db, db
	common.RedisEnabled = false
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.PasswordLoginEnabled = true
	common.LoginFailureRateLimitEnable = true
	common.LoginFailureRateLimitNum = 1
	common.LoginFailureIPRateLimitNum = 10
	common.LoginFailureRateLimitDuration = 60
	common.LoginInflightIPLimit = 8
	common.LoginInflightLeaseDuration = 10
	common.SessionSecret = "controller-login-rate-limit-secret"
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedis
		common.SetMainDatabaseType(previousDatabaseType)
		common.PasswordLoginEnabled = previousPasswordLoginEnabled
		common.LoginFailureRateLimitEnable = previousRateLimitEnabled
		common.LoginFailureRateLimitNum = previousAccountLimit
		common.LoginFailureIPRateLimitNum = previousIPLimit
		common.LoginFailureRateLimitDuration = previousDuration
		common.LoginInflightIPLimit = previousInflightLimit
		common.LoginInflightLeaseDuration = previousLeaseDuration
		common.SessionSecret = previousSessionSecret
		_ = sqlDB.Close()
	})
	return db
}

func performPasswordLogin(t *testing.T, engine http.Handler, username, password, clientIP string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	request := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = clientIP + ":12345"
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

func performPasswordLoginRaw(t *testing.T, engine http.Handler, body, clientIP string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = clientIP + ":12345"
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

func decodePasswordLoginResponse(t *testing.T, recorder *httptest.ResponseRecorder) (bool, string, bool) {
	t.Helper()
	var response struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Data    struct {
			RequireTwoFA bool `json:"require_2fa"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response.Success, response.Code, response.Data.RequireTwoFA
}

func TestPasswordLoginCountsFailuresButNotSuccess(t *testing.T) {
	db := setupPasswordLoginRateLimitDB(t)
	hashedPassword, err := common.Password2Hash("correct-password")
	require.NoError(t, err)
	user := &model.User{
		Username: "login-rate-user", Password: hashedPassword, Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
	}
	require.NoError(t, db.Create(user).Error)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/api/user/login", Login)
	clientIP := "192.0.2.181"

	for range 2 {
		response := performPasswordLogin(t, engine, user.Username, "correct-password", clientIP)
		require.Equal(t, http.StatusOK, response.Code)
		success, code, requireTwoFA := decodePasswordLoginResponse(t, response)
		assert.True(t, success)
		assert.Empty(t, code)
		assert.False(t, requireTwoFA)
	}

	failed := performPasswordLogin(t, engine, user.Username, "wrong-password", clientIP)
	require.Equal(t, http.StatusOK, failed.Code)
	success, _, _ := decodePasswordLoginResponse(t, failed)
	assert.False(t, success)

	limited := performPasswordLogin(t, engine, user.Username, "wrong-password", clientIP)
	require.Equal(t, http.StatusTooManyRequests, limited.Code)
	_, code, _ := decodePasswordLoginResponse(t, limited)
	assert.Equal(t, "AUTH_LOGIN_RATE_LIMITED", code)
}

func TestPasswordLoginTwoFAChallengeDoesNotCountAsFailure(t *testing.T) {
	require.NoError(t, appI18n.Init())
	db := setupPasswordLoginRateLimitDB(t)
	hashedPassword, err := common.Password2Hash("correct-password")
	require.NoError(t, err)
	user := &model.User{
		Username: "login-rate-twofa", Password: hashedPassword, Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
	}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&model.TwoFA{UserId: user.Id, Secret: "secret", IsEnabled: true}).Error)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/api/user/login", Login)
	for range 2 {
		response := performPasswordLogin(t, engine, user.Username, "correct-password", "192.0.2.182")
		require.Equal(t, http.StatusOK, response.Code)
		success, code, requireTwoFA := decodePasswordLoginResponse(t, response)
		assert.True(t, success)
		assert.Empty(t, code)
		assert.True(t, requireTwoFA)
	}
}

func TestMalformedLoginPayloadDoesNotConsumePartialUsernameBucket(t *testing.T) {
	db := setupPasswordLoginRateLimitDB(t)
	common.LoginFailureRateLimitNum = 5
	hashedPassword, err := common.Password2Hash("correct-password")
	require.NoError(t, err)
	user := &model.User{
		Username: "victim", Password: hashedPassword, Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
	}
	require.NoError(t, db.Create(user).Error)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/api/user/login", Login)
	clientIP := "192.0.2.184"
	for range 5 {
		response := performPasswordLoginRaw(t, engine, `{"username":"victim","password":123}`, clientIP)
		require.Equal(t, http.StatusOK, response.Code)
		success, _, _ := decodePasswordLoginResponse(t, response)
		assert.False(t, success)
	}

	valid := performPasswordLogin(t, engine, "victim", "correct-password", clientIP)
	require.Equal(t, http.StatusOK, valid.Code)
	success, code, _ := decodePasswordLoginResponse(t, valid)
	assert.True(t, success)
	assert.Empty(t, code)
}
