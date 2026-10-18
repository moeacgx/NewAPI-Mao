package controller

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/extension"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func marketplaceArchiveFixture(t *testing.T, id, version string, host extension.HostCompat) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	manifest := extension.Manifest{ID: id, Name: "市场测试模块", Version: version, Host: host, Runtime: extension.Runtime{Type: extension.RuntimeTypeStatic, StaticDir: "public"}}
	data, err := common.Marshal(manifest)
	require.NoError(t, err)
	for name, content := range map[string][]byte{"manifest.json": data, "public/index.html": []byte("<p>版本 " + version + "</p>")} {
		file, err := writer.Create(name)
		require.NoError(t, err)
		_, err = file.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func marketplaceUploadRequest(t *testing.T, archive []byte, values map[string][]string) *http.Request {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	file, err := writer.CreateFormFile("file", "module.zip")
	require.NoError(t, err)
	_, err = file.Write(archive)
	require.NoError(t, err)
	for field, items := range values {
		for _, item := range items {
			require.NoError(t, writer.WriteField(field, item))
		}
	}
	require.NoError(t, writer.Close())
	request := httptest.NewRequest(http.MethodPost, "/api/extension-admin/upload", bytes.NewReader(buffer.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestExtensionMarketplaceUploadMetadataValidation(t *testing.T) {
	valid := map[string][]string{
		"archiveSha256": {strings.Repeat("a", 64)}, "expectedId": {"market-demo"}, "expectedVersion": {"2.0.0"},
		"catalogUrl": {ExtensionMarketplaceCatalogURL}, "archivePath": {"published/market-demo/2.0.0/market-demo-2.0.0.zip"},
	}
	for _, test := range []struct {
		name, field, value string
		remove, duplicate  bool
	}{
		{name: "摘要格式", field: "archiveSha256", value: "not-sha256"},
		{name: "摘要长度", field: "archiveSha256", value: strings.Repeat("a", 63)},
		{name: "身份路径逃逸", field: "expectedId", value: "../market-demo"},
		{name: "版本路径逃逸", field: "expectedVersion", value: "2/0"},
		{name: "版本双点", field: "expectedVersion", value: "1..2"},
		{name: "来源变更", field: "catalogUrl", value: "https://example.com/catalog.json"},
		{name: "来源查询", field: "catalogUrl", value: ExtensionMarketplaceCatalogURL + "?x=1"},
		{name: "绝对路径", field: "archivePath", value: "https://example.com/module.zip"},
		{name: "目录穿越", field: "archivePath", value: "published/market-demo/2.0.0/../market-demo-2.0.0.zip"},
		{name: "编码路径", field: "archivePath", value: "published/market-demo/2.0.0/%6darket-demo-2.0.0.zip"},
		{name: "查询参数", field: "archivePath", value: "published/market-demo/2.0.0/market-demo-2.0.0.zip?a=1"},
		{name: "片段", field: "archivePath", value: "published/market-demo/2.0.0/market-demo-2.0.0.zip#part"},
		{name: "反斜杠", field: "archivePath", value: `published\market-demo\2.0.0\market-demo-2.0.0.zip`},
		{name: "错误版本路径", field: "archivePath", value: "published/market-demo/1.0.0/market-demo-1.0.0.zip"},
		{name: "缺字段", field: "expectedVersion", remove: true},
		{name: "空字段", field: "catalogUrl", value: ""},
		{name: "重复字段", field: "expectedId", duplicate: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			values := make(map[string][]string, len(valid))
			for field, items := range valid {
				values[field] = append([]string(nil), items...)
			}
			if test.remove {
				delete(values, test.field)
			} else if test.duplicate {
				values[test.field] = append(values[test.field], values[test.field][0])
			} else {
				values[test.field] = []string{test.value}
			}
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
			ctx.Request.MultipartForm = &multipart.Form{Value: values}
			_, err := extensionMarketplaceUploadExpectation(ctx)
			require.Error(t, err)
		})
	}
	for _, values := range []map[string][]string{nil, valid} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		ctx.Request.MultipartForm = &multipart.Form{Value: values}
		expectation, err := extensionMarketplaceUploadExpectation(ctx)
		require.NoError(t, err)
		assert.Equal(t, values == nil, expectation == nil)
	}
}

func TestExtensionMarketplaceUploadRejectsMismatchWithoutReplacingEnabledModule(t *testing.T) {
	previousManager, previousVersion := extension.DefaultManager, common.Version
	common.Version = "v1.0.0"
	t.Cleanup(func() { extension.DefaultManager, common.Version = previousManager, previousVersion })
	for _, test := range []struct {
		name, hash, expectedID, expectedVersion, catalog, path string
		host                                                   extension.HostCompat
	}{
		{name: "字节摘要不同", hash: strings.Repeat("0", 64)},
		{name: "真实身份不同", expectedID: "other-module"},
		{name: "真实版本不同", expectedVersion: "3.0.0"},
		{name: "固定来源不同", catalog: "http://127.0.0.1:1/catalog.json"},
		{name: "归档路径不同", path: "published/wrong.zip"},
		{name: "宿主不兼容", host: extension.HostCompat{Min: "v9.0.0"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager := extension.NewManager(t.TempDir())
			extension.DefaultManager = manager
			oldArchive := marketplaceArchiveFixture(t, "market-demo", "1.0.0", extension.HostCompat{})
			_, err := manager.InstallArchive(bytes.NewReader(oldArchive), int64(len(oldArchive)))
			require.NoError(t, err)
			_, err = manager.SetEnabled("market-demo", true)
			require.NoError(t, err)
			beforeFile, err := os.ReadFile(filepath.Join(manager.RootDir(), "market-demo", "public/index.html"))
			require.NoError(t, err)
			beforeState, err := os.ReadFile(filepath.Join(manager.RootDir(), "state.json"))
			require.NoError(t, err)
			archive := marketplaceArchiveFixture(t, "market-demo", "2.0.0", test.host)
			id, version := "market-demo", "2.0.0"
			if test.expectedID != "" {
				id = test.expectedID
			}
			if test.expectedVersion != "" {
				version = test.expectedVersion
			}
			values := map[string][]string{"archiveSha256": {fmt.Sprintf("%x", sha256.Sum256(archive))}, "expectedId": {id}, "expectedVersion": {version}, "catalogUrl": {ExtensionMarketplaceCatalogURL}, "archivePath": {fmt.Sprintf("published/%s/%s/%s-%s.zip", id, version, id, version)}}
			if test.hash != "" {
				values["archiveSha256"] = []string{test.hash}
			}
			if test.catalog != "" {
				values["catalogUrl"] = []string{test.catalog}
			}
			if test.path != "" {
				values["archivePath"] = []string{test.path}
			}
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = marketplaceUploadRequest(t, archive, values)
			ctx.Set("role", common.RoleRootUser)
			UploadExtension(ctx)
			require.Contains(t, recorder.Body.String(), `"success":false`)
			afterFile, err := os.ReadFile(filepath.Join(manager.RootDir(), "market-demo", "public/index.html"))
			require.NoError(t, err)
			afterState, err := os.ReadFile(filepath.Join(manager.RootDir(), "state.json"))
			require.NoError(t, err)
			assert.Equal(t, beforeFile, afterFile)
			assert.Equal(t, beforeState, afterState)
			installed, ok := manager.Get("market-demo")
			require.True(t, ok)
			assert.True(t, installed.Enabled)
			assert.Equal(t, "1.0.0", installed.Version)
		})
	}
}

func TestExtensionMarketplaceUploadRejectsArchiveAboveLimit(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("role", common.RoleRootUser)
	ctx.Request = marketplaceUploadRequest(t, []byte("file-content"), nil)
	require.NoError(t, ctx.Request.ParseMultipartForm(1024))
	ctx.Request.MultipartForm.File["file"][0].Size = extension.MaxInstallArchiveBytes + 1
	UploadExtension(ctx)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.Contains(t, recorder.Body.String(), "too large")
}

func TestExtensionMarketplaceUploadRejectsMetadataFilePartsWithoutReplacingInstalled(t *testing.T) {
	previousManager := extension.DefaultManager
	t.Cleanup(func() { extension.DefaultManager = previousManager })
	for _, mixed := range []bool{false, true} {
		name := "五项全为文件字段"
		if mixed {
			name = "文本与文件字段混合"
		}
		t.Run(name, func(t *testing.T) {
			manager := extension.NewManager(t.TempDir())
			extension.DefaultManager = manager
			oldArchive := marketplaceArchiveFixture(t, "market-demo", "1.0.0", extension.HostCompat{})
			_, err := manager.InstallArchive(bytes.NewReader(oldArchive), int64(len(oldArchive)))
			require.NoError(t, err)
			_, err = manager.SetEnabled("market-demo", true)
			require.NoError(t, err)
			oldFile, err := os.ReadFile(filepath.Join(manager.RootDir(), "market-demo", "public/index.html"))
			require.NoError(t, err)
			oldState, err := os.ReadFile(filepath.Join(manager.RootDir(), "state.json"))
			require.NoError(t, err)
			archive := marketplaceArchiveFixture(t, "market-demo", "2.0.0", extension.HostCompat{})
			values := map[string]string{"archiveSha256": fmt.Sprintf("%x", sha256.Sum256(archive)), "expectedId": "market-demo", "expectedVersion": "2.0.0", "catalogUrl": ExtensionMarketplaceCatalogURL, "archivePath": "published/market-demo/2.0.0/market-demo-2.0.0.zip"}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			file, err := writer.CreateFormFile("file", "module.zip")
			require.NoError(t, err)
			_, err = file.Write(archive)
			require.NoError(t, err)
			for field, value := range values {
				if mixed {
					require.NoError(t, writer.WriteField(field, value))
				}
				if mixed && field != "expectedId" {
					continue
				}
				part, err := writer.CreateFormFile(field, field+".txt")
				require.NoError(t, err)
				_, err = part.Write([]byte(value))
				require.NoError(t, err)
			}
			require.NoError(t, writer.Close())
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/extension-admin/upload", bytes.NewReader(body.Bytes()))
			ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
			ctx.Set("role", common.RoleRootUser)
			UploadExtension(ctx)
			require.Contains(t, recorder.Body.String(), `"success":false`)
			afterFile, err := os.ReadFile(filepath.Join(manager.RootDir(), "market-demo", "public/index.html"))
			require.NoError(t, err)
			afterState, err := os.ReadFile(filepath.Join(manager.RootDir(), "state.json"))
			require.NoError(t, err)
			assert.Equal(t, oldFile, afterFile)
			assert.Equal(t, oldState, afterState)
			installed, ok := manager.Get("market-demo")
			require.True(t, ok)
			assert.True(t, installed.Enabled)
			assert.Equal(t, "1.0.0", installed.Version)
		})
	}
}
