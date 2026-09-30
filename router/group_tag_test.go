package router

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupTagRoutesRequireAdminAndSupportLifecycle(t *testing.T) {
	_, admin := setupSecurityAuditRouterTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.GroupTag{}, &model.GroupTagBinding{}))
	user := model.User{Id: 503, Username: "tag-reader", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1}
	require.NoError(t, model.DB.Create(&user).Error)
	adminAuth := securityAuditAuthorization(t, admin.Id)
	userAuth := securityAuditAuthorization(t, user.Id)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	tag, err := model.SaveGroupTag(model.GroupTag{Name: "待编辑分类", GroupIDs: []int{1}})
	require.NoError(t, err)
	path := fmt.Sprintf("/api/group/tags/%d", tag.Id)
	for _, auth := range []string{"", userAuth} {
		for _, route := range []struct{ method, path string }{
			{http.MethodGet, "/api/group/tags"},
			{http.MethodPost, "/api/group/tags"},
			{http.MethodPut, path},
			{http.MethodDelete, path},
		} {
			request := httptest.NewRequest(route.method, route.path, bytes.NewBufferString(`{"name":"越权修改"}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", auth)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			var response struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.False(t, response.Success, "%s %s", route.method, route.path)
		}
	}
	stored, err := model.GetGroupTags(nil)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, "待编辑分类", stored[0].Name)

	for _, route := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/group/tags", ""},
		{http.MethodPut, path, `{"name":"管理员修改","icons":["OpenAI","Claude.Color"],"group_ids":[1]}`},
		{http.MethodPost, "/api/group/tags", `{"name":"新标签","group_ids":[]}`},
		{http.MethodDelete, path, ""},
	} {
		request := httptest.NewRequest(route.method, route.path, bytes.NewBufferString(route.body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", adminAuth)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		var response struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success, "%s %s: %s", route.method, route.path, response.Message)
	}
	stored, err = model.GetGroupTags(nil)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, "新标签", stored[0].Name)
	var bindings int64
	require.NoError(t, model.DB.Model(&model.GroupTagBinding{}).Count(&bindings).Error)
	assert.Zero(t, bindings)
}
