package controller

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationPreviewRendersCustomEmojiAndQuoteWithoutSending(t *testing.T) {
	setupNotificationControllerTestDB(t)
	body, err := common.Marshal(map[string]any{
		"event_type": model.NotificationEventTypeTicketUserMessage,
		"template":   "<b>🎫 #{{ticket_id}}</b>\n👤 {{user_id}}\n<blockquote>{{content}}</blockquote>\n{{mention}}",
		"target":     map[string]string{"mention_user_id": "123", "mention_name": "客服 <Ops>"},
	})
	require.NoError(t, err)
	response := invokeNotificationHandler(PreviewNotificationTemplate, common.RoleRootUser, http.MethodPost, "/api/notification/preview", string(body), 0)
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			HTML   string `json:"html"`
			Length int    `json:"text_length"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
	require.True(t, result.Success, response.Body.String())
	assert.Contains(t, result.Data.HTML, "<b>🎫 #13</b>\n👤 6494\n<blockquote>")
	assert.Contains(t, result.Data.HTML, `<a href="tg://user?id=123">客服 &lt;Ops&gt;</a>`)
	assert.NotContains(t, result.Data.HTML, "正文：")
	assert.Positive(t, result.Data.Length)
	var events, deliveries int64
	require.NoError(t, model.DB.Model(&model.NotificationEvent{}).Count(&events).Error)
	require.NoError(t, model.DB.Model(&model.NotificationDelivery{}).Count(&deliveries).Error)
	assert.Zero(t, events)
	assert.Zero(t, deliveries)
}

func TestNotificationPreviewAndSaveRejectSameInvalidTemplate(t *testing.T) {
	setupNotificationControllerTestDB(t)
	bot := model.NotificationBot{Name: "测试", Token: "test-token", Enabled: true}
	require.NoError(t, model.CreateNotificationBot(&bot))
	for _, template := range []string{
		"<script>alert(1)</script>",
		`<a href="https://example.com/{{title}}">查看</a>`,
		"<b><i>错位</b></i>",
		"{{unknown}}",
		"{{content}}{{content}}{{content}}",
		strings.Repeat("字", 4091),
	} {
		t.Run(fmt.Sprintf("模板_%d", len(template)), func(t *testing.T) {
			req := notificationTaskRequest{Name: "测试", EventType: model.NotificationEventTypeTicketUserMessage, BotId: bot.Id, Template: template, Targets: []notificationTargetRequest{{ChatId: "-1001"}}}
			require.Error(t, validateNotificationTaskRequest(&req))
			body, err := common.Marshal(map[string]string{"event_type": req.EventType, "template": template})
			require.NoError(t, err)
			response := invokeNotificationHandler(PreviewNotificationTemplate, common.RoleRootUser, http.MethodPost, "/api/notification/preview", string(body), 0)
			assert.Contains(t, response.Body.String(), `"success":false`)
		})
	}
}

func TestNotificationPreviewRequiresRootAndBoundsInput(t *testing.T) {
	response := invokeNotificationHandler(PreviewNotificationTemplate, common.RoleAdminUser, http.MethodPost, "/api/notification/preview", `{}`, 0)
	assert.Contains(t, response.Body.String(), `"message":"no permission"`)
	body := `{"event_type":"ticket_user_message","template":"` + strings.Repeat("a", 65*1024) + `"}`
	response = invokeNotificationHandler(PreviewNotificationTemplate, common.RoleRootUser, http.MethodPost, "/api/notification/preview", body, 0)
	assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
}
