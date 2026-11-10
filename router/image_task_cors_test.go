package router

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageTaskBrowserCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := newCORSTestEngine()
	SetRelayRouter(engine)
	setWebRouter(engine, static.LocalFile(t.TempDir(), false), []byte("frontend fallback"))

	for _, test := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "create", method: http.MethodPost, path: "/v1/images/tasks"},
		{name: "edit", method: http.MethodPost, path: "/v1/images/tasks?action=edits"},
		{name: "poll", method: http.MethodGet, path: "/v1/images/tasks/cors-task"},
		{name: "content", method: http.MethodGet, path: "/v1/images/tasks/cors-task/content/0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			preflight := servePreflight(engine, test.path, "https://browser.example.test", test.method, "authorization,content-type")
			require.Equal(t, http.StatusNoContent, preflight.Code)
			assert.Empty(t, preflight.Body.String())
			assert.Equal(t, "*", preflight.Header().Get("Access-Control-Allow-Origin"))
			assert.Empty(t, preflight.Header().Get("Access-Control-Allow-Credentials"))
			methods := strings.Split(preflight.Header().Get("Access-Control-Allow-Methods"), ",")
			assert.Contains(t, methods, http.MethodGet)
			assert.Contains(t, methods, http.MethodPost)
			assert.Contains(t, methods, http.MethodOptions)
			headers := strings.Split(strings.ToLower(preflight.Header().Get("Access-Control-Allow-Headers")), ",")
			assert.Contains(t, headers, "authorization")
			assert.Contains(t, headers, "content-type")

			request := httptest.NewRequest(test.method, "https://api.example.test"+test.path, nil)
			request.Header.Set("Origin", "https://browser.example.test")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			require.Equal(t, http.StatusUnauthorized, response.Code)
			assert.Equal(t, "*", response.Header().Get("Access-Control-Allow-Origin"))
			assert.Empty(t, response.Header().Get("Access-Control-Allow-Credentials"))
			assert.Contains(t, response.Header().Get("Content-Type"), gin.MIMEJSON)
			assert.Contains(t, response.Body.String(), `"error"`)
		})
	}
}

func TestImageTaskBrowserAuthenticatedReads(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupFeatureRouterAuthTest(t)
	model.InitDBColumns()
	t.Cleanup(model.InitDBColumns)
	require.NoError(t, db.AutoMigrate(&model.Token{}, &model.TokenGroupBinding{}))
	owner, _, ownerToken := newTokenQueryUser(t, common.UserStatusEnabled)
	_, _, otherToken := newTokenQueryUser(t, common.UserStatusEnabled)
	previousRetention := common.GetImageTaskDataRetentionHours()
	common.SetImageTaskDataRetentionHours(1)
	t.Cleanup(func() { common.SetImageTaskDataRetentionHours(previousRetention) })

	image := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	taskData, err := common.Marshal(gin.H{"data": []gin.H{{"b64_json": base64.StdEncoding.EncodeToString(image)}}})
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.Task{
		TaskID: "cors-completed-image", UserId: owner.Id, Group: "default",
		Platform: constant.TaskPlatformImage, Status: model.TaskStatusSuccess,
		FinishTime: time.Now().Unix(), Data: taskData,
	}).Error)
	require.NoError(t, db.Create(&model.Task{
		TaskID: "cors-canvas-image", UserId: owner.Id, Group: "default",
		Platform: constant.TaskPlatformCanvasImage, Status: model.TaskStatusSuccess,
		FinishTime: time.Now().Unix(), Data: taskData,
	}).Error)

	engine := newCORSTestEngine()
	SetRelayRouter(engine)
	for _, test := range []struct {
		name   string
		path   string
		key    string
		status int
	}{
		{name: "owner polls", path: "/v1/images/tasks/cors-completed-image", key: ownerToken.Key, status: http.StatusOK},
		{name: "owner reads PNG", path: "/v1/images/tasks/cors-completed-image/content/0", key: ownerToken.Key, status: http.StatusOK},
		{name: "other user cannot poll", path: "/v1/images/tasks/cors-completed-image", key: otherToken.Key, status: http.StatusNotFound},
		{name: "other user cannot read PNG", path: "/v1/images/tasks/cors-completed-image/content/0", key: otherToken.Key, status: http.StatusNotFound},
		{name: "other platform cannot poll", path: "/v1/images/tasks/cors-canvas-image", key: ownerToken.Key, status: http.StatusNotFound},
		{name: "other platform cannot read PNG", path: "/v1/images/tasks/cors-canvas-image/content/0", key: ownerToken.Key, status: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "https://api.example.test"+test.path, nil)
			request.Header.Set("Origin", "https://browser.example.test")
			request.Header.Set("Authorization", "Bearer sk-"+test.key)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			require.Equal(t, test.status, response.Code, response.Body.String())
			assert.Equal(t, "*", response.Header().Get("Access-Control-Allow-Origin"))
			assert.Empty(t, response.Header().Get("Access-Control-Allow-Credentials"))
			if test.status != http.StatusOK {
				assert.NotContains(t, response.Body.String(), "b64_json")
				assert.NotContains(t, response.Body.String(), "cors-completed-image")
				assert.NotContains(t, response.Body.String(), "cors-canvas-image")
				return
			}
			if strings.HasSuffix(test.path, "/content/0") {
				assert.Equal(t, "image/png", response.Header().Get("Content-Type"))
				assert.Equal(t, image, response.Body.Bytes())
				return
			}
			var payload map[string]any
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			assert.Equal(t, "succeeded", payload["status"])
			assert.Contains(t, response.Body.String(), "/v1/images/tasks/cors-completed-image/content/0")
		})
	}
}
