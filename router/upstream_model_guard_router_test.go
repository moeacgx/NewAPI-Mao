package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUpstreamModelGuardAPITest(t *testing.T) (*gin.Engine, string, string, <-chan struct{}) {
	t.Helper()
	root, admin := setupSecurityAuditRouterTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.UpstreamModelGuardConfig{}, &model.UpstreamModelGuardRecord{}, &model.AutoGroupMember{}, &model.Log{}))
	auditWritten := make(chan struct{}, 32)
	require.NoError(t, model.DB.Callback().Create().After("gorm:commit_or_rollback_transaction").Register("guard-test:audit-written", func(tx *gorm.DB) {
		if tx.Statement.Table == "logs" {
			auditWritten <- struct{}{}
		}
	}))
	originalRateLimit, originalCriticalLimit := common.GlobalApiRateLimitEnable, common.CriticalRateLimitEnable
	common.GlobalApiRateLimitEnable, common.CriticalRateLimitEnable = false, false
	t.Cleanup(func() {
		common.GlobalApiRateLimitEnable, common.CriticalRateLimitEnable = originalRateLimit, originalCriticalLimit
	})
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	return engine, securityAuditAuthorization(t, root.Id), securityAuditAuthorization(t, admin.Id), auditWritten
}

func awaitUpstreamModelGuardAudit(t *testing.T, auditWritten <-chan struct{}) {
	t.Helper()
	select {
	case <-auditWritten:
	case <-time.After(5 * time.Second):
		t.Fatal("管理操作审计未完成")
	}
}

func TestUpstreamModelGuardRoutesRequireRootAndPersistVersionedRules(t *testing.T) {
	engine, rootAuth, adminAuth, auditWritten := setupUpstreamModelGuardAPITest(t)
	for _, endpoint := range []string{"config", "groups", "records"} {
		for _, test := range []struct {
			auth   string
			status int
		}{{"", http.StatusUnauthorized}, {adminAuth, http.StatusForbidden}, {rootAuth, http.StatusOK}} {
			request := httptest.NewRequest(http.MethodGet, "/api/extensions/upstream-model-guard/"+endpoint, nil)
			request.Header.Set("Authorization", test.auth)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			assert.Equal(t, test.status, recorder.Code, recorder.Body.String())
			assert.Contains(t, recorder.Header().Get("Cache-Control"), "no-store")
		}
	}
	for _, test := range []struct {
		auth   string
		status int
	}{{"", http.StatusUnauthorized}, {adminAuth, http.StatusForbidden}} {
		request := httptest.NewRequest(http.MethodPut, "/api/extensions/upstream-model-guard/config", strings.NewReader(`{}`))
		request.Header.Set("Authorization", test.auth)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		assert.Equal(t, test.status, recorder.Code)
		assert.Contains(t, recorder.Header().Get("Cache-Control"), "no-store")
	}
	require.NoError(t, model.DB.Create(&model.Group{Code: "vip", Name: "VIP", Status: model.GroupStatusActive}).Error)
	body := `{"expected_version":1,"enabled":true,"rules":[{"enabled":true,"group_codes":["default","vip"],"model":"client-model","upstream_models":["provider-model","provider-version"]},{"enabled":true,"group_codes":["vip"],"model":"client-second","upstream_models":["provider-second"]}]}`
	for _, expected := range []int{http.StatusOK, http.StatusConflict} {
		request := httptest.NewRequest(http.MethodPut, "/api/extensions/upstream-model-guard/config", strings.NewReader(body))
		request.Header.Set("Authorization", rootAuth)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		awaitUpstreamModelGuardAudit(t, auditWritten)
		require.Equal(t, expected, recorder.Code, recorder.Body.String())
	}
	config, err := model.LoadUpstreamModelGuardConfig(t.Context())
	require.NoError(t, err)
	assert.EqualValues(t, 2, config.ConfigVersion)
	assert.True(t, config.Enabled)
	assert.Contains(t, config.RulesJSON, "provider-version")
	request := httptest.NewRequest(http.MethodGet, "/api/extensions/upstream-model-guard/config", nil)
	request.Header.Set("Authorization", rootAuth)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	var response struct {
		Success bool                             `json:"success"`
		Data    service.UpstreamModelGuardConfig `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data.Rules, 2)
	assert.Equal(t, []string{"default", "vip"}, response.Data.Rules[0].GroupCodes)
	assert.Equal(t, []string{"provider-model", "provider-version"}, response.Data.Rules[0].UpstreamModels)
	assert.Equal(t, "client-second", response.Data.Rules[1].Model)
}

func TestUpstreamModelGuardRecordsRejectInvalidPagination(t *testing.T) {
	engine, rootAuth, _, _ := setupUpstreamModelGuardAPITest(t)
	request := httptest.NewRequest(http.MethodGet, "/api/extensions/upstream-model-guard/records?page=-1&page_size=100000", nil)
	request.Header.Set("Authorization", rootAuth)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestUpstreamModelGuardConfigRejectsInvalidRulesAndAllowsEmptyDisabledConfig(t *testing.T) {
	engine, rootAuth, _, auditWritten := setupUpstreamModelGuardAPITest(t)
	for _, body := range []string{
		`{`,
		`{"expected_version":0,"enabled":true,"rules":[]}`,
		`{"expected_version":1,"enabled":true,"rules":[{}]}`,
		`{"expected_version":1,"enabled":true,"rules":[{"enabled":true,"group_codes":[],"model":"client","upstream_models":["provider"]}]}`,
		`{"expected_version":1,"enabled":true,"rules":[{"enabled":true,"group_codes":["missing"],"model":"client","upstream_models":["provider"]}]}`,
		`{"expected_version":1,"enabled":true,"rules":[{"enabled":true,"group_codes":["default"],"model":"client","upstream_models":[]}]}`,
		`{"expected_version":1,"enabled":true,"rules":[{"enabled":true,"group_codes":["default"],"model":"client","upstream_models":["provider"]},{"enabled":true,"group_codes":["default"],"model":"client","upstream_models":["other"]}]}`,
	} {
		request := httptest.NewRequest(http.MethodPut, "/api/extensions/upstream-model-guard/config", strings.NewReader(body))
		request.Header.Set("Authorization", rootAuth)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		awaitUpstreamModelGuardAudit(t, auditWritten)
		assert.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	request := httptest.NewRequest(http.MethodPut, "/api/extensions/upstream-model-guard/config", strings.NewReader(`{"expected_version":1,"enabled":false,"rules":[]}`))
	request.Header.Set("Authorization", rootAuth)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	awaitUpstreamModelGuardAudit(t, auditWritten)
	assert.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
}

func TestUpstreamModelGuardRecordsReturnDescendingPage(t *testing.T) {
	engine, rootAuth, _, _ := setupUpstreamModelGuardAPITest(t)
	for index := 1; index <= 3; index++ {
		require.NoError(t, model.DB.Create(&model.UpstreamModelGuardRecord{ChannelID: index, ChannelName: "channel", GroupID: 1, Group: "default", RequestedModel: "client", ExpectedUpstreamModelsJSON: `["provider"]`, ActualUpstreamModel: "wrong", ConfigVersion: 1}).Error)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/extensions/upstream-model-guard/records?page=2&page_size=2", nil)
	request.Header.Set("Authorization", rootAuth)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Items    []model.UpstreamModelGuardRecord `json:"items"`
			Total    int64                            `json:"total"`
			Page     int                              `json:"page"`
			PageSize int                              `json:"page_size"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, int64(3), response.Data.Total)
	assert.Equal(t, 2, response.Data.Page)
	assert.Equal(t, 2, response.Data.PageSize)
	require.Len(t, response.Data.Items, 1)
	assert.Equal(t, 1, response.Data.Items[0].ChannelID)
	assert.Equal(t, []string{"provider"}, response.Data.Items[0].ExpectedUpstreamModels)
}
