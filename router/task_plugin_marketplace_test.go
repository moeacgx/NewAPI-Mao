package router

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskPluginMarketplaceSourcePermissionsAndClear(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFeatureRouterAuthTest(t)
	model.InitDBColumns()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	oldLogDB := model.LOG_DB
	model.LOG_DB = db
	t.Cleanup(func() { model.LOG_DB = oldLogDB })
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.TaskPlugin{}, &model.Log{}))
	require.NoError(t, service.InitTaskPlugins())
	_, admin := issueFeatureRouterSession(t, common.RoleAdminUser, "mkt-admin")
	_, root := issueFeatureRouterSession(t, common.RoleRootUser, "mkt-root")
	_, user := issueFeatureRouterSession(t, common.RoleCommonUser, "mkt-user")
	engine := gin.New()
	registerTaskPluginManagement(engine.Group("/api"))
	require.NotEqual(t, http.StatusOK, serveFeatureRouterRequest(engine, http.MethodGet, "/api/plugin/task/marketplace/sources", user).Code)
	rec := serveFeatureRouterRequest(engine, http.MethodGet, "/api/plugin/task/marketplace/sources", admin)
	require.Equal(t, http.StatusOK, rec.Code)
	req := httptest.NewRequest(http.MethodPut, "/api/plugin/task/marketplace/sources", bytes.NewBufferString("[]"))
	req.Header.Set("Authorization", "Bearer "+root)
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	rec = serveFeatureRouterRequest(engine, http.MethodGet, "/api/plugin/task/marketplace/sources", admin)
	require.Contains(t, rec.Body.String(), `"data":[]`)
}

func TestTaskPluginMarketplaceInstallContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFeatureRouterAuthTest(t)
	model.InitDBColumns()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.TaskPlugin{}, &model.Channel{}, &model.Log{}))
	oldLogDB := model.LOG_DB
	model.LOG_DB = db
	t.Cleanup(func() { model.LOG_DB = oldLogDB })
	_, root := issueFeatureRouterSession(t, common.RoleRootUser, "market-install-root")
	_, admin := issueFeatureRouterSession(t, common.RoleAdminUser, "market-install-admin")
	engine := gin.New()
	// 本用例验证真实鉴权与控制器。错误路径的异步通用审计另有测试，避免它跨用例访问全局数据库。
	engine.Use(func(c *gin.Context) { common.SetContextKey(c, constant.ContextKeyAuditLogged, true) })
	registerTaskPluginManagement(engine.Group("/api"))
	call := func(method, path, token string, body any) *httptest.ResponseRecorder {
		payload, err := common.Marshal(body)
		require.NoError(t, err)
		req := httptest.NewRequest(method, path, bytes.NewReader(payload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		return rec
	}
	sourceConfig := []setting.TaskPluginMarketplaceSource{{Name: "Maintained", IndexURL: "https://plugins.example/stable/index.json"}}
	assert.Equal(t, http.StatusForbidden, call(http.MethodPut, "/api/plugin/task/marketplace/sources", admin, sourceConfig).Code)
	require.Equal(t, http.StatusOK, call(http.MethodPut, "/api/plugin/task/marketplace/sources", root, sourceConfig).Code)
	assert.Equal(t, http.StatusBadRequest, call(http.MethodPut, "/api/plugin/task/marketplace/sources", root, nil).Code)
	source := `export const meta={apiVersion:1,key:"market-demo",name:"Marketplace Demo",version:"1.0.0",author:{name:"Test"},models:["test-model"],fetchMode:"per_task"};
export function buildSubmitRequest(){return {};}
export function parseSubmitResponse(){return {};}
export function buildQueryRequest(){return {};}
export function parseTaskResult(){return {};}`
	provenance := map[string]string{"name": "Maintained", "index_url": sourceConfig[0].IndexURL, "path": "published/market-demo/1.0.0/plugin.js"}
	body := map[string]any{"source": source, "sourceSha256": fmt.Sprintf("%x", sha256.Sum256([]byte(source))), "expectedKey": "market-demo", "expectedVersion": "1.0.0", "remark": "reviewed", "marketplace": provenance}
	assert.Equal(t, http.StatusForbidden, call(http.MethodPost, "/api/plugin/task", admin, body).Code)
	rec := call(http.MethodPost, "/api/plugin/task", root, body)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var response struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &response))
	require.True(t, response.Success, rec.Body.String())
	assert.Equal(t, false, response.Data["active"])
	assert.Equal(t, false, response.Data["enabled"])
	assert.Equal(t, "custom", response.Data["source_kind"])
	assert.IsType(t, map[string]any{}, response.Data["marketplace"])
	row, err := model.GetTaskPluginVersion("market-demo", "1.0.0")
	require.NoError(t, err)
	assert.Contains(t, row.Marketplace, "published/market-demo/1.0.0/plugin.js")
	require.NoError(t, model.ActivateTaskPlugin(row.Key, row.Version))
	body["remark"] = "must-not-overwrite"
	require.Equal(t, http.StatusOK, call(http.MethodPost, "/api/plugin/task", root, body).Code)
	reloaded, err := model.GetTaskPluginVersion(row.Key, row.Version)
	require.NoError(t, err)
	assert.True(t, reloaded.Active)
	assert.True(t, reloaded.Enabled)
	assert.Equal(t, "reviewed", reloaded.Remark)
	assert.Equal(t, row.Marketplace, reloaded.Marketplace)
	for _, tc := range []struct {
		name, field string
		value       any
		status      int
		code        string
	}{
		{"hash", "sourceSha256", strings.Repeat("0", 64), 400, "marketplace_hash_mismatch"},
		{"key", "expectedKey", "different", 400, "marketplace_meta_mismatch"},
		{"version", "expectedVersion", "2.0.0", 400, "marketplace_meta_mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := body[tc.field]
			body[tc.field] = tc.value
			defer func() { body[tc.field] = old }()
			rec := call(http.MethodPost, "/api/plugin/task", root, body)
			assert.Equal(t, tc.status, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.code)
		})
	}
	for _, path := range []string{"../private.js", "%2e%2e%2fprivate.js", "dir%5cprivate.js", "dir/%252e%252e/private.js", "dir/%00x.js", "dir/%20x.js", "dir//x.js"} {
		provenance["path"] = path
		rec := call(http.MethodPost, "/api/plugin/task", root, body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, path)
		assert.Contains(t, rec.Body.String(), "marketplace_path_invalid")
	}
	provenance["path"] = "published/market-demo/1.0.0/plugin.js"
	body["source"] = source + "\n// changed"
	body["sourceSha256"] = fmt.Sprintf("%x", sha256.Sum256([]byte(body["source"].(string))))
	assert.Equal(t, http.StatusConflict, call(http.MethodPost, "/api/plugin/task", root, body).Code)
	body["source"], body["sourceSha256"] = source, row.SourceHash
	require.Equal(t, http.StatusOK, call(http.MethodPut, "/api/plugin/task/marketplace/sources", root, []setting.TaskPluginMarketplaceSource{}).Code)
	assert.Equal(t, http.StatusBadRequest, call(http.MethodPost, "/api/plugin/task", root, body).Code)
	_, err = service.LoadPinnedTaskPlugin(&model.TaskPluginSnapshot{Key: row.Key, Version: row.Version, SourceHash: row.SourceHash, SourceKind: row.SourceKind, APIVersion: row.APIVersion})
	require.NoError(t, err)
	versions := serveFeatureRouterRequest(engine, http.MethodGet, "/api/plugin/task/market-demo/versions", admin)
	assert.Contains(t, versions.Body.String(), `"marketplace":{`)
	assert.Contains(t, versions.Body.String(), `"id":`)
	assert.NotContains(t, versions.Body.String(), `"source":`)
	manual := strings.Replace(source, "market-demo", "manual-demo", 1)
	require.Equal(t, http.StatusOK, call(http.MethodPost, "/api/plugin/task", root, map[string]string{"source": manual}).Code)
	invalid := call(http.MethodPost, "/api/plugin/task", root, map[string]string{"source": "throw new Error('FAKE_SECRET_MARKER')"})
	assert.Equal(t, http.StatusBadRequest, invalid.Code)
	assert.NotContains(t, invalid.Body.String(), "FAKE_SECRET_MARKER")
	assert.Equal(t, http.StatusBadRequest, call(http.MethodPost, "/api/plugin/task", root, map[string]string{"source": manual, "sourceSha256": strings.Repeat("0", 64)}).Code)
	assert.Equal(t, http.StatusRequestEntityTooLarge, call(http.MethodPost, "/api/plugin/task", root, map[string]string{"source": strings.Repeat("a", (1<<20)+1)}).Code)
}

func TestTaskPluginUploadBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFeatureRouterAuthTest(t)
	model.InitDBColumns()
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.TaskPlugin{}))
	require.NoError(t, service.InitTaskPlugins())
	_, root := issueFeatureRouterSession(t, common.RoleRootUser, "mkt-limit")
	engine := gin.New()
	// 请求体边界用例不测试通用异步审计。
	engine.Use(func(c *gin.Context) { common.SetContextKey(c, constant.ContextKeyAuditLogged, true) })
	registerTaskPluginManagement(engine.Group("/api"))
	req := httptest.NewRequest(http.MethodPost, "/api/plugin/task", bytes.NewBufferString(`{"source":"`+strings.Repeat("a", 8<<20)+`"}`))
	req.Header.Set("Authorization", "Bearer "+root)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}
