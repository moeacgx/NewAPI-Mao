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
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			response = managementRequest(fixture.engine, method, "/api/prompt-center/admin/", credential, "192.0.2.180", `{}`)
			assert.Equal(t, http.StatusForbidden, response.Code)
		}
		response = managementRequest(fixture.engine, http.MethodPut, "/api/prompt-center/admin/1", credential, "192.0.2.180", `{}`)
		assert.Equal(t, http.StatusForbidden, response.Code)
	}
	response := managementRequest(fixture.engine, http.MethodGet, "/api/prompt-center/admin/", fixture.owner.pat, "192.0.2.180", "")
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"success":true`)
}
