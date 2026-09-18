package router

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/extension"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtensionMarketplaceRouteRequiresRootAndReturnsFixedSource(t *testing.T) {
	db := setupFeatureRouterAuthTest(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	_, user := issueFeatureRouterSession(t, common.RoleCommonUser, "extension-market-user")
	_, admin := issueFeatureRouterSession(t, common.RoleAdminUser, "extension-market-admin")
	_, root := issueFeatureRouterSession(t, common.RoleRootUser, "extension-market-root")
	engine := gin.New()
	engine.Use(func(c *gin.Context) { common.SetContextKey(c, constant.ContextKeyAuditLogged, true) })
	registerExtensionRoutes(engine.Group("/api"))
	assert.Equal(t, http.StatusUnauthorized, serveFeatureRouterRequest(engine, http.MethodGet, "/api/extension-admin/marketplace", "").Code)
	for _, token := range []string{user, admin} {
		assert.Equal(t, http.StatusForbidden, serveFeatureRouterRequest(engine, http.MethodGet, "/api/extension-admin/marketplace", token).Code)
		assert.Equal(t, http.StatusForbidden, serveFeatureRouterRequest(engine, http.MethodPost, "/api/extension-admin/upload", token).Code)
	}
	recorder := serveFeatureRouterRequest(engine, http.MethodGet, "/api/extension-admin/marketplace", root)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	assert.Equal(t, map[string]any{"catalog_url": controller.ExtensionMarketplaceCatalogURL, "repository_url": controller.ExtensionMarketplaceRepositoryURL, "host_version": common.Version, "max_archive_bytes": float64(extension.MaxInstallArchiveBytes)}, response.Data)
	assert.Contains(t, recorder.Header().Get("Cache-Control"), "no-store")
}

func TestExtensionMarketplaceRouteInstallsRealArchiveAndKeepsManualUpload(t *testing.T) {
	db := setupFeatureRouterAuthTest(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	_, root := issueFeatureRouterSession(t, common.RoleRootUser, "extension-market-install")
	engine := gin.New()
	engine.Use(func(c *gin.Context) { common.SetContextKey(c, constant.ContextKeyAuditLogged, true) })
	registerExtensionRoutes(engine.Group("/api"))
	for _, marketplace := range []bool{true, false} {
		id := "marketplace-installed"
		if !marketplace {
			id = "manual-installed"
		}
		manifest := extension.Manifest{ID: id, Name: "路由上传模块", Version: "1.2.3", Runtime: extension.Runtime{Type: extension.RuntimeTypeStatic, StaticDir: "public"}}
		manifestData, err := common.Marshal(manifest)
		require.NoError(t, err)
		var archive bytes.Buffer
		zipWriter := zip.NewWriter(&archive)
		for name, data := range map[string][]byte{"manifest.json": manifestData, "public/index.html": []byte("<p>合法模块</p>")} {
			file, err := zipWriter.Create(name)
			require.NoError(t, err)
			_, err = file.Write(data)
			require.NoError(t, err)
		}
		require.NoError(t, zipWriter.Close())
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		file, err := writer.CreateFormFile("file", "module.zip")
		require.NoError(t, err)
		_, err = file.Write(archive.Bytes())
		require.NoError(t, err)
		if marketplace {
			for name, value := range map[string]string{"archiveSha256": fmt.Sprintf("%x", sha256.Sum256(archive.Bytes())), "expectedId": id, "expectedVersion": manifest.Version, "catalogUrl": controller.ExtensionMarketplaceCatalogURL, "archivePath": "published/" + id + "/1.2.3/" + id + "-1.2.3.zip"} {
				require.NoError(t, writer.WriteField(name, value))
			}
		}
		require.NoError(t, writer.Close())
		request := httptest.NewRequest(http.MethodPost, "/api/extension-admin/upload", bytes.NewReader(body.Bytes()))
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Set("Authorization", "Bearer "+root)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Contains(t, recorder.Body.String(), `"success":true`, recorder.Body.String())
		installed, ok := extension.DefaultManager.Get(id)
		require.True(t, ok)
		assert.Equal(t, manifest.Version, installed.Version)
		assert.False(t, installed.Enabled)
	}
}
