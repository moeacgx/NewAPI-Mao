package router

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptCenterRoutesRequireAdminAndKeepUserOwnership(t *testing.T) {
	fixture := setupTokenKeyFixture(t, false, common.RoleRootUser, 100)
	require.NoError(t, fixture.db.AutoMigrate(&model.PromptTemplateGroup{}, &model.TokenGroupBinding{}))
	fixture.engine = newManagementRouter(t)
	for _, credential := range []string{fixture.peer.pat, fixture.peer.session} {
		response := managementRequest(fixture.engine, http.MethodGet, "/api/prompt-center/", credential, "192.0.2.180", "")
		assert.Equal(t, http.StatusOK, response.Code)
		assert.Contains(t, response.Body.String(), `"success":true`)
		response = managementRequest(fixture.engine, http.MethodGet, "/api/prompt-center/categories/", credential, "192.0.2.180", "")
		assert.Equal(t, http.StatusOK, response.Code)
		assert.Contains(t, response.Body.String(), `"success":true`)
		for _, route := range []struct{ method, path string }{{http.MethodPost, "/api/prompt-center/admin/categories/"}, {http.MethodPut, "/api/prompt-center/admin/categories/1"}} {
			response = managementRequest(fixture.engine, route.method, route.path, credential, "192.0.2.180", `{"name":"不允许"}`)
			assert.Equal(t, http.StatusForbidden, response.Code)
		}
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			response = managementRequest(fixture.engine, method, "/api/prompt-center/admin/", credential, "192.0.2.180", `{}`)
			assert.Equal(t, http.StatusForbidden, response.Code)
		}
		for _, route := range []struct{ method, path string }{
			{http.MethodGet, "/api/prompt-center/admin/store/"},
			{http.MethodPost, "/api/prompt-center/admin/store/bundled:code-review/import"},
			{http.MethodPost, "/api/prompt-center/admin/import/archive"},
			{http.MethodPost, "/api/prompt-center/admin/import/repository"},
		} {
			response = managementRequest(fixture.engine, route.method, route.path, credential, "192.0.2.180", `{}`)
			assert.Equal(t, http.StatusForbidden, response.Code)
		}
		response = managementRequest(fixture.engine, http.MethodPut, "/api/prompt-center/admin/1", credential, "192.0.2.180", `{}`)
		assert.Equal(t, http.StatusForbidden, response.Code)
	}
	response := managementRequest(fixture.engine, http.MethodGet, "/api/prompt-center/admin/", fixture.owner.pat, "192.0.2.180", "")
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"success":true`)
	response = managementRequest(fixture.engine, http.MethodGet, "/api/prompt-center/admin/store/", fixture.owner.pat, "192.0.2.180", "")
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"source":"bundled"`)
}
