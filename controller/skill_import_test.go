package controller

import (
	"archive/zip"
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillArchiveAPIProducesDraftWithoutPersistence(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.PromptTemplate{}))
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	file, err := w.Create("review/SKILL.md")
	require.NoError(t, err)
	_, err = file.Write([]byte("---\nname: 审查助手\ndescription: 检查边界条件\n---\n仔细审查输入。"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	var body bytes.Buffer
	upload := multipart.NewWriter(&body)
	part, err := upload.CreateFormFile("file", "review.zip")
	require.NoError(t, err)
	_, err = part.Write(archive.Bytes())
	require.NoError(t, err)
	require.NoError(t, upload.Close())
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", &body)
	ctx.Request.Header.Set("Content-Type", upload.FormDataContentType())
	ImportSkillArchive(ctx)
	var response struct {
		Success bool                 `json:"success"`
		Data    model.PromptTemplate `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, recorder.Body.String())
	assert.Equal(t, "审查助手", response.Data.Title)
	assert.Equal(t, "仔细审查输入。", response.Data.Content)
	assert.Zero(t, response.Data.ID)
	assert.Zero(t, response.Data.CategoryID)
	assert.False(t, response.Data.Enabled)
	assert.Empty(t, response.Data.GroupIDs)
	var count int64
	require.NoError(t, db.Model(&model.PromptTemplate{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestSkillArchiveAPIRejectsOversizedAndDuplicateUploads(t *testing.T) {
	for _, test := range []struct {
		name  string
		size  int
		files int
	}{{"超大文件", service.MaxSkillArchiveBytes + 1, 1}, {"多个文件", 1, 2}} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			upload := multipart.NewWriter(&body)
			for i := 0; i < test.files; i++ {
				part, err := upload.CreateFormFile("file", "test.zip")
				require.NoError(t, err)
				_, err = part.Write([]byte(strings.Repeat("x", test.size)))
				require.NoError(t, err)
			}
			require.NoError(t, upload.Close())
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/", &body)
			ctx.Request.Header.Set("Content-Type", upload.FormDataContentType())
			ImportSkillArchive(ctx)
			assert.False(t, decodeAPIResponse(t, recorder).Success)
		})
	}
}

func TestSkillStoreAPIImportsSeparateUnpublishedCopy(t *testing.T) {
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/", nil, 1)
	GetSkillStore(ctx)
	var catalog struct {
		Success bool                        `json:"success"`
		Data    []service.SkillStoreSummary `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &catalog))
	require.True(t, catalog.Success)
	require.NotEmpty(t, catalog.Data)
	assert.NotContains(t, recorder.Body.String(), `"content"`)
	ctx, recorder = newAuthenticatedContext(t, http.MethodPost, "/", nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: catalog.Data[0].ID}}
	ImportStoreSkill(ctx)
	var response struct {
		Success bool                 `json:"success"`
		Data    model.PromptTemplate `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, recorder.Body.String())
	assert.NotEmpty(t, response.Data.Content)
	assert.Zero(t, response.Data.ID)
	assert.False(t, response.Data.Enabled)
	assert.Empty(t, response.Data.GroupIDs)
	assert.Zero(t, response.Data.CategoryID)
}

func TestRepositoryImportAPIRejectsMalformedAndNonGitHubURLs(t *testing.T) {
	for _, body := range []string{`{`, `{"url":"https://127.0.0.1/private"}`, `{"url":"https://github.com/"}`, `{"url":"` + strings.Repeat("a", 4096) + `"}`} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		ImportRepositorySkills(ctx)
		assert.False(t, decodeAPIResponse(t, recorder).Success)
	}
}

func TestRepositoryImportAPIValidatesEntireJSONBody(t *testing.T) {
	for _, body := range []string{
		`{"url":"https://github.com/acme/skills"}` + strings.Repeat(" ", 4096),
		`{"url":"https://github.com/acme/skills"}{"url":"ignored"}`,
	} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		ImportRepositorySkills(ctx)
		response := decodeAPIResponse(t, recorder)
		assert.False(t, response.Success)
		assert.Contains(t, response.Message, "请求格式错误或超过大小限制")
	}
}
