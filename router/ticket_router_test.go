package router

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTicketRoutesProtectAdminAndPersonalData(t *testing.T) {
	fixture := setupTokenKeyFixture(t, false, common.RoleAdminUser, 1)
	common.GlobalApiRateLimitEnable = false
	common.CriticalRateLimitEnable = false
	wasMaster := common.IsMasterNode
	common.IsMasterNode = true
	t.Cleanup(func() { common.IsMasterNode = wasMaster })
	require.NoError(t, fixture.db.AutoMigrate(&model.Ticket{}, &model.TicketMessage{}, &model.CasbinRule{}, &model.AuthzRole{}))
	require.NoError(t, authz.Init(fixture.db))
	fixture.engine = newManagementRouter(t)
	owned, err := model.CreateTicket(fixture.owner.user.Id, model.TicketCreateInput{Title: "管理员自己的工单", Category: "general", Priority: "normal", Content: "个人问题"})
	require.NoError(t, err)
	peer, err := model.CreateTicket(fixture.peer.user.Id, model.TicketCreateInput{Title: "其他用户的工单", Category: "technical", Priority: "normal", Content: "他人问题"})
	require.NoError(t, err)
	for _, tc := range []struct {
		name, credential string
		status           int
	}{
		{"未登录", "", http.StatusUnauthorized},
		{"普通用户PAT", fixture.peer.pat, http.StatusForbidden},
		{"普通用户会话", fixture.peer.session, http.StatusForbidden},
		{"管理员PAT", fixture.owner.pat, http.StatusOK},
		{"管理员会话", fixture.owner.session, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := managementRequest(fixture.engine, http.MethodGet, "/api/ticket-admin", tc.credential, "192.0.2.44", "")
			require.Equal(t, tc.status, r.Code, r.Body.String())
			if tc.status == http.StatusOK {
				assert.Contains(t, r.Header().Get("Cache-Control"), "no-store")
				assert.Contains(t, r.Body.String(), peer.Title)
				assert.Contains(t, r.Body.String(), owned.Title)
				assert.NotContains(t, r.Body.String(), "他人问题")
			}
		})
	}
	r := managementRequest(fixture.engine, http.MethodGet, fmt.Sprintf("/api/tickets?user_id=%d", fixture.peer.user.Id), fixture.owner.session, "192.0.2.44", "")
	require.Equal(t, http.StatusOK, r.Code)
	assert.Contains(t, r.Body.String(), owned.Title)
	assert.NotContains(t, r.Body.String(), peer.Title)
	r = managementRequest(fixture.engine, http.MethodGet, fmt.Sprintf("/api/tickets/%d", peer.Id), fixture.owner.session, "192.0.2.44", "")
	assert.Equal(t, http.StatusNotFound, r.Code)

	// 单独撤销管理写权限后仍能读，但不能回复或改状态。
	require.NoError(t, authz.SetUserPermissions(fixture.owner.user.Id, authz.PermissionsMap{"tickets": {"read": true, "write": false}}))
	r = managementRequest(fixture.engine, http.MethodGet, "/api/ticket-admin", fixture.owner.session, "192.0.2.44", "")
	assert.Equal(t, http.StatusOK, r.Code)
	for _, route := range []struct{ method, path, body string }{
		{http.MethodPost, fmt.Sprintf("/api/ticket-admin/%d/messages", peer.Id), `{"content":"无权限回复"}`},
		{http.MethodPut, fmt.Sprintf("/api/ticket-admin/%d/status", peer.Id), `{"status":"closed"}`},
	} {
		r = managementRequest(fixture.engine, route.method, route.path, fixture.owner.session, "192.0.2.44", route.body)
		assert.Equal(t, http.StatusForbidden, r.Code, r.Body.String())
	}
	require.NoError(t, authz.SetUserPermissions(fixture.owner.user.Id, authz.PermissionsMap{"tickets": {"read": false, "write": false}}))
	r = managementRequest(fixture.engine, http.MethodGet, "/api/ticket-admin", fixture.owner.session, "192.0.2.44", "")
	assert.Equal(t, http.StatusForbidden, r.Code)
	r = managementRequest(fixture.engine, http.MethodGet, "/api/tickets", fixture.owner.session, "192.0.2.44", "")
	assert.Equal(t, http.StatusOK, r.Code)
	require.NoError(t, authz.ClearUserPermissions(fixture.owner.user.Id))
}

func TestTicketBatchAndAutomationRoutesEnforcePermissionsAndVersions(t *testing.T) {
	fixture := setupTokenKeyFixture(t, false, common.RoleAdminUser, 1)
	wasMaster := common.IsMasterNode
	common.IsMasterNode = true
	t.Cleanup(func() { common.IsMasterNode = wasMaster })
	oldGlobal, oldCritical := common.GlobalApiRateLimitEnable, common.CriticalRateLimitEnable
	common.GlobalApiRateLimitEnable = false
	common.CriticalRateLimitEnable = false
	t.Cleanup(func() { common.GlobalApiRateLimitEnable = oldGlobal; common.CriticalRateLimitEnable = oldCritical })
	model.InitDBColumns()
	t.Cleanup(model.InitDBColumns)
	require.NoError(t, fixture.db.AutoMigrate(&model.Ticket{}, &model.TicketMessage{}, &model.TicketActionLog{}, &model.Option{}, &model.CasbinRule{}, &model.AuthzRole{}))
	require.NoError(t, authz.Init(fixture.db))
	fixture.engine = newManagementRouter(t)
	ticket, err := model.CreateTicket(fixture.peer.user.Id, model.TicketCreateInput{Title: "批量权限", Category: "general", Priority: "normal", Content: "保留正文"})
	require.NoError(t, err)
	refs := fmt.Sprintf(`"tickets":[{"id":%d,"version":1}]`, ticket.Id)
	routes := []struct{ method, path, body string }{
		{http.MethodGet, "/api/ticket-admin/automation/auto-reply", ""},
		{http.MethodGet, "/api/ticket-admin/automation/auto-archive", ""},
		{http.MethodPut, "/api/ticket-admin/automation/auto-reply", `{"enabled":false,"content":""}`},
		{http.MethodPut, "/api/ticket-admin/automation/auto-archive", `{"enabled":false,"inactive_days":30,"statuses":["resolved","closed"]}`},
		{http.MethodPut, "/api/ticket-admin/batch/status", "{" + refs + `,"status":"archived"}`},
		{http.MethodPost, "/api/ticket-admin/batch/delete", "{" + refs + "}"},
	}
	for _, route := range routes {
		for _, cred := range []struct {
			token  string
			status int
		}{{"", 401}, {fixture.peer.pat, 403}, {fixture.peer.session, 403}} {
			response := managementRequest(fixture.engine, route.method, route.path, cred.token, "192.0.2.44", route.body)
			require.Equal(t, cred.status, response.Code, response.Body.String())
		}
	}
	require.NoError(t, authz.SetUserPermissions(fixture.owner.user.Id, authz.PermissionsMap{"tickets": {"read": true, "write": false}}))
	for _, route := range routes {
		expected := http.StatusForbidden
		if route.method == http.MethodGet {
			expected = http.StatusOK
		}
		for _, credential := range []string{fixture.owner.session, fixture.owner.pat} {
			response := managementRequest(fixture.engine, route.method, route.path, credential, "192.0.2.44", route.body)
			require.Equal(t, expected, response.Code, response.Body.String())
		}
	}
	require.NoError(t, authz.ClearUserPermissions(fixture.owner.user.Id))
	response := managementRequest(fixture.engine, http.MethodPut, "/api/ticket-admin/batch/status", fixture.owner.session, "192.0.2.44", "{"+refs+`,"status":"archived"}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), `"updated_count":1`)
	response = managementRequest(fixture.engine, http.MethodPost, "/api/ticket-admin/batch/delete", fixture.owner.session, "192.0.2.44", "{"+refs+"}")
	require.Equal(t, 409, response.Code, response.Body.String())
	refs = fmt.Sprintf(`"tickets":[{"id":%d,"version":2}]`, ticket.Id)
	response = managementRequest(fixture.engine, http.MethodPost, "/api/ticket-admin/batch/delete", fixture.owner.session, "192.0.2.44", "{"+refs+"}")
	require.Equal(t, 200, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), `"deleted_count":1`)
	for _, path := range []string{fmt.Sprintf("/api/tickets/%d", ticket.Id), fmt.Sprintf("/api/ticket-admin/%d", ticket.Id)} {
		response = managementRequest(fixture.engine, http.MethodGet, path, fixture.owner.session, "192.0.2.44", "")
		assert.Equal(t, 404, response.Code)
	}
	var stored model.Ticket
	assert.ErrorIs(t, fixture.db.First(&stored, ticket.Id).Error, gorm.ErrRecordNotFound)
	require.NoError(t, fixture.db.Unscoped().First(&stored, ticket.Id).Error)
	for _, body := range []string{`{"enabled":true,"inactive_days":0,"statuses":["open"]}`, `{"enabled":true,"inactive_days":1.5,"statuses":["open"]}`, `{"enabled":true,"inactive_days":1,"statuses":[]}`} {
		response = managementRequest(fixture.engine, http.MethodPut, "/api/ticket-admin/automation/auto-archive", fixture.owner.session, "192.0.2.44", body)
		assert.Equal(t, 400, response.Code, response.Body.String())
	}
	response = managementRequest(fixture.engine, http.MethodPut, "/api/ticket-admin/automation/auto-archive", fixture.owner.session, "192.0.2.44", `{"enabled":true,"inactive_days":7,"statuses":["open"]}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), `"inactive_days":7`)
}

func TestTicketAdminStatsRouteRequiresAdminReadPermission(t *testing.T) {
	fixture := setupTokenKeyFixture(t, false, common.RoleAdminUser, 1)
	common.GlobalApiRateLimitEnable = false
	wasMaster := common.IsMasterNode
	common.IsMasterNode = true
	t.Cleanup(func() { common.IsMasterNode = wasMaster })
	require.NoError(t, fixture.db.AutoMigrate(&model.Ticket{}, &model.TicketMessage{}, &model.CasbinRule{}, &model.AuthzRole{}))
	require.NoError(t, authz.Init(fixture.db))
	fixture.engine = newManagementRouter(t)
	_, err := model.CreateTicket(fixture.peer.user.Id, model.TicketCreateInput{Title: "其他用户的工单", Category: "technical", Priority: "normal", Content: "他人问题"})
	require.NoError(t, err)

	for _, tc := range []struct {
		name, credential string
		status           int
	}{
		{"未登录", "", http.StatusUnauthorized},
		{"普通用户会话", fixture.peer.session, http.StatusForbidden},
		{"管理员会话", fixture.owner.session, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := managementRequest(fixture.engine, http.MethodGet, "/api/ticket-admin/stats", tc.credential, "192.0.2.44", "")
			require.Equal(t, tc.status, r.Code, r.Body.String())
			if tc.status == http.StatusOK {
				assert.Contains(t, r.Header().Get("Cache-Control"), "no-store")
				assert.JSONEq(t, `{"success":true,"data":{"open":1,"processing":0,"resolved":0,"closed":0,"archived":0}}`, r.Body.String())
			}
		})
	}

	require.NoError(t, authz.SetUserPermissions(fixture.owner.user.Id, authz.PermissionsMap{"tickets": {"read": false, "write": false}}))
	r := managementRequest(fixture.engine, http.MethodGet, "/api/ticket-admin/stats", fixture.owner.session, "192.0.2.44", "")
	assert.Equal(t, http.StatusForbidden, r.Code, r.Body.String())
}
