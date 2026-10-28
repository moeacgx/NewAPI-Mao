package controller

import (
	"bytes"
	"fmt"
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

func ticketHandlerRequest(handler gin.HandlerFunc, userID, role int, method, path, body string, id int) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", userID)
	c.Set("role", role)
	c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if id > 0 {
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	}
	handler(c)
	return recorder
}

func TestTicketControllersEnforceOwnershipAndIgnoreSpoofedIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupNotificationControllerTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Ticket{}, &model.TicketMessage{}))
	api := TicketAPI{}
	admin := TicketAPI{Admin: true}
	created := ticketHandlerRequest(api.Create, 10, common.RoleAdminUser, http.MethodPost, "/api/tickets", `{"title":"工单","category":"general","priority":"normal","content":"正文","user_id":20,"author_role":"admin","status":"resolved"}`, 0)
	require.Equal(t, http.StatusOK, created.Code, created.Body.String())
	var response struct{ Data model.Ticket }
	require.NoError(t, common.Unmarshal(created.Body.Bytes(), &response))
	assert.Equal(t, 10, response.Data.UserId)
	assert.Equal(t, model.TicketStatusOpen, response.Data.Status)
	ticketID := response.Data.Id
	for _, handler := range []gin.HandlerFunc{api.Detail, api.Reply, api.Status} {
		denied := ticketHandlerRequest(handler, 20, common.RoleCommonUser, http.MethodPost, "/api/tickets/1", `{"content":"越权","status":"closed"}`, ticketID)
		assert.Equal(t, http.StatusNotFound, denied.Code, denied.Body.String())
	}
	blocked := ticketHandlerRequest(admin.List, 20, common.RoleCommonUser, http.MethodGet, "/api/ticket-admin", "", 0)
	assert.Equal(t, http.StatusForbidden, blocked.Code)
	reply := ticketHandlerRequest(api.Reply, 10, common.RoleAdminUser, http.MethodPost, "/api/tickets/1/messages", `{"content":"管理员自己的追问"}`, ticketID)
	require.Equal(t, http.StatusOK, reply.Code, reply.Body.String())
	assert.Contains(t, reply.Body.String(), `"author_role":"user"`)
	adminReply := ticketHandlerRequest(admin.Reply, 99, common.RoleAdminUser, http.MethodPost, "/api/ticket-admin/1/messages", `{"content":"客服回复"}`, ticketID)
	require.Equal(t, http.StatusOK, adminReply.Code, adminReply.Body.String())
	assert.Contains(t, adminReply.Body.String(), `"author_role":"admin"`)
	list := ticketHandlerRequest(api.List, 10, common.RoleAdminUser, http.MethodGet, "/api/tickets?user_id=20", "", 0)
	require.Equal(t, http.StatusOK, list.Code)
	assert.Contains(t, list.Body.String(), `"user_id":10`)
}

func TestTicketControllersRejectInvalidPaginationAndOversizeRequests(t *testing.T) {
	api := TicketAPI{}
	for _, query := range []string{"p=-1", "p=no", "p=1000001", "page_size=-1", "page_size=no", "page_size=0", "status=invalid", "keyword=" + strings.Repeat("a", 121)} {
		t.Run(query, func(t *testing.T) {
			r := ticketHandlerRequest(api.List, 10, common.RoleCommonUser, http.MethodGet, "/api/tickets?"+query, "", 0)
			assert.Equal(t, http.StatusBadRequest, r.Code, r.Body.String())
		})
	}
	for _, query := range []string{"before_message_id=-1", "before_message_id=no"} {
		r := ticketHandlerRequest(api.Detail, 10, common.RoleCommonUser, http.MethodGet, "/api/tickets/1?"+query, "", 1)
		assert.Equal(t, http.StatusBadRequest, r.Code)
	}
	large := `{"content":"` + strings.Repeat("a", 65536) + `"}`
	r := ticketHandlerRequest(api.Reply, 10, common.RoleCommonUser, http.MethodPost, "/api/tickets/1/messages", large, 1)
	assert.Equal(t, http.StatusRequestEntityTooLarge, r.Code)
	r = ticketHandlerRequest(api.Create, 0, 0, http.MethodPost, "/api/tickets", `{}`, 0)
	assert.Equal(t, http.StatusUnauthorized, r.Code)
}

func TestTicketNotificationEventsRegisterAndValidateTemplates(t *testing.T) {
	for _, legacy := range []string{"ticket_created", "ticket_user_replied", "ticket_admin_replied", "ticket_status_changed"} {
		_, ok := findNotificationEventDefinition(legacy)
		assert.False(t, ok, "已合并或下线的工单事件不再提供给通知任务选择: %s", legacy)
	}
	for _, event := range []string{model.NotificationEventTypeTicketUserMessage} {
		t.Run(event, func(t *testing.T) {
			definition, ok := findNotificationEventDefinition(event)
			require.True(t, ok)
			assert.True(t, definition.Available)
			assert.Equal(t, "core", definition.Owner)
			assert.Contains(t, definition.Variables, "content")
			assert.Contains(t, definition.Variables, "action_label")
			assert.Equal(t, definition.DefaultTemplate, model.NormalizeNotificationTaskTemplate(event, ""))
			assert.Equal(t, definition.DefaultTemplate, model.NormalizeNotificationTaskTemplate(event, model.NotificationTaskDefaultTemplate))
			definition.SamplePayload["title"] = "<b>用户输入</b>"
			definition.SamplePayload["content"] = "请开票 <b>不是标签</b> & 保留换行\n第二行"
			rendered, err := service.RenderNotificationTemplate(definition.DefaultTemplate, definition.SamplePayload, nil)
			require.NoError(t, err)
			assert.Contains(t, rendered, "&lt;b&gt;用户输入&lt;/b&gt;")
			assert.Contains(t, rendered, "用户追加回复")
			assert.Contains(t, rendered, "正文：请开票 &lt;b&gt;不是标签&lt;/b&gt; &amp; 保留换行\n第二行")
			body, err := service.RenderNotificationTemplate("{{content}}", definition.SamplePayload, nil)
			require.NoError(t, err)
			assert.Equal(t, "请开票 &lt;b&gt;不是标签&lt;/b&gt; &amp; 保留换行\n第二行", body)
			_, err = service.RenderNotificationTemplate("{{unknown}}", definition.SamplePayload, nil)
			assert.Error(t, err)
		})
	}
}

func TestTicketRequestRejectsOversizeTrailingDataAndMultipleJSONValues(t *testing.T) {
	setupNotificationControllerTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Ticket{}, &model.TicketMessage{}))
	body := `{"title":"标题","category":"general","priority":"normal","content":"正文"}`
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"超长尾随空白", body + strings.Repeat(" ", 65536), http.StatusRequestEntityTooLarge},
		{"多个JSON对象", body + `{"title":"第二个对象"}`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Set("id", 10)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/tickets", strings.NewReader(tc.body))
			c.Request.ContentLength = -1
			c.Request.Header.Set("Content-Type", "application/json")
			TicketAPI{}.Create(c)
			assert.Equal(t, tc.status, recorder.Code, recorder.Body.String())
		})
	}
	var count int64
	require.NoError(t, model.DB.Model(&model.Ticket{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestTicketAutoReplyControllerRequiresAdminAndPersistsConfig(t *testing.T) {
	setupNotificationControllerTestDB(t)
	model.InitDBColumns()
	t.Cleanup(model.InitDBColumns)
	require.NoError(t, model.DB.AutoMigrate(&model.Option{}))
	admin := TicketAPI{Admin: true}
	user := TicketAPI{}

	denied := ticketHandlerRequest(admin.AutoReply, 10, common.RoleCommonUser, http.MethodGet, "/api/ticket-admin/automation/auto-reply", "", 0)
	assert.Equal(t, http.StatusForbidden, denied.Code)
	defaultConfig := ticketHandlerRequest(admin.AutoReply, 10, common.RoleAdminUser, http.MethodGet, "/api/ticket-admin/automation/auto-reply", "", 0)
	require.Equal(t, http.StatusOK, defaultConfig.Code, defaultConfig.Body.String())
	assert.Contains(t, defaultConfig.Body.String(), `"enabled":false`)

	invalid := ticketHandlerRequest(admin.UpdateAutoReply, 10, common.RoleAdminUser, http.MethodPut, "/api/ticket-admin/automation/auto-reply", `{"enabled":true,"content":""}`, 0)
	assert.Equal(t, http.StatusBadRequest, invalid.Code, invalid.Body.String())
	saved := ticketHandlerRequest(admin.UpdateAutoReply, 10, common.RoleAdminUser, http.MethodPut, "/api/ticket-admin/automation/auto-reply", `{"enabled":true,"content":"  已收到，我们会尽快处理。  "}`, 0)
	require.Equal(t, http.StatusOK, saved.Code, saved.Body.String())
	assert.Contains(t, saved.Body.String(), `"enabled":true`)
	assert.Contains(t, saved.Body.String(), `已收到，我们会尽快处理。`)

	personalDenied := ticketHandlerRequest(user.AutoReply, 10, common.RoleAdminUser, http.MethodGet, "/api/ticket-admin/automation/auto-reply", "", 0)
	assert.Equal(t, http.StatusForbidden, personalDenied.Code, personalDenied.Body.String())
}

func TestTicketNotificationTasksAcceptContentAndRejectUnknownVariables(t *testing.T) {
	setupNotificationControllerTestDB(t)
	bot := &model.NotificationBot{Name: "工单测试机器人", Token: "fixture-token", Enabled: true}
	require.NoError(t, model.CreateNotificationBot(bot))
	for _, legacy := range []string{"ticket_created", "ticket_admin_replied", "ticket_status_changed"} {
		body := map[string]any{"name": legacy, "event_type": legacy, "bot_id": bot.Id, "enabled": true, "template": "", "targets": []map[string]any{{"chat_id": "123", "enabled": true}}}
		encoded, err := common.Marshal(body)
		require.NoError(t, err)
		rejected := invokeNotificationHandler(CreateNotificationTask, common.RoleRootUser, http.MethodPost, "/api/notification/tasks", string(encoded), 0)
		assert.Contains(t, rejected.Body.String(), `"success":false`, legacy)
	}
	for _, event := range []string{model.NotificationEventTypeTicketUserMessage} {
		t.Run(event, func(t *testing.T) {
			body := map[string]any{"name": event, "event_type": event, "bot_id": bot.Id, "enabled": true, "template": "", "targets": []map[string]any{{"chat_id": "123", "enabled": true}}}
			encoded, err := common.Marshal(body)
			require.NoError(t, err)
			saved := invokeNotificationHandler(CreateNotificationTask, common.RoleRootUser, http.MethodPost, "/api/notification/tasks", string(encoded), 0)
			require.Contains(t, saved.Body.String(), `"success":true`)
			var task model.NotificationTask
			require.NoError(t, model.DB.Where("event_type = ?", event).First(&task).Error)
			assert.Equal(t, model.NormalizeNotificationTaskTemplate(event, ""), task.Template)
			body["template"] = "{{content}}"
			encoded, err = common.Marshal(body)
			require.NoError(t, err)
			withContent := invokeNotificationHandler(CreateNotificationTask, common.RoleRootUser, http.MethodPost, "/api/notification/tasks", string(encoded), 0)
			require.Contains(t, withContent.Body.String(), `"success":true`)
			for _, template := range []string{
				"{{unknown}}",
				strings.Repeat("{{content}}", 5),
				"{{content}}{{content}}" + strings.Repeat("{{ticket_id}}", 20),
				strings.Repeat("文", 1800) + "{{mention}}{{content}}",
			} {
				body["template"] = template
				encoded, err = common.Marshal(body)
				require.NoError(t, err)
				rejected := invokeNotificationHandler(CreateNotificationTask, common.RoleRootUser, http.MethodPost, "/api/notification/tasks", string(encoded), 0)
				assert.Contains(t, rejected.Body.String(), `"success":false`, template)
			}
			var count int64
			require.NoError(t, model.DB.Model(&model.NotificationTask{}).Where("event_type = ?", event).Count(&count).Error)
			assert.EqualValues(t, 2, count)
		})
	}
}
