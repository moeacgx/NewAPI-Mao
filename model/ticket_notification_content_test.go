package model

import (
	"html"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicketNotificationsSnapshotCurrentUserContent(t *testing.T) {
	setupTicketTestDB(t)
	subscribeTicketEvents(t)
	config, err := common.Marshal(TicketAutoReplyConfig{Enabled: true, Content: "系统自动回复不应进入通知"})
	require.NoError(t, err)
	require.NoError(t, DB.Create(&Option{Key: TicketAutoReplyOptionKey, Value: string(config)}).Error)
	ticket, err := CreateTicket(10, TicketCreateInput{
		Title: "开发票问题", Category: "billing", Priority: "normal", Content: "用户首条 <b>正文</b> & 😺",
	})
	require.NoError(t, err)
	_, err = ReplyTicket(ticket.Id, 10, false, "用户追加 <script>正文</script>\n第二行 &🙂")
	require.NoError(t, err)
	_, err = ReplyTicket(ticket.Id, 99, true, "管理员正文不应进入通知")
	require.NoError(t, err)
	_, err = ChangeTicketStatus(ticket.Id, 99, true, TicketStatusResolved)
	require.NoError(t, err)
	_, err = ReplyTicket(ticket.Id, 10, false, "另一条用户回复")
	require.NoError(t, err)

	var events []NotificationEvent
	require.NoError(t, DB.Order("id").Find(&events).Error)
	require.Len(t, events, 3)
	wantContent := []string{"用户首条 <b>正文</b> & 😺", "用户追加 <script>正文</script>\n第二行 &🙂", "另一条用户回复"}
	for index, event := range events {
		var payload map[string]any
		require.NoError(t, common.UnmarshalJsonStr(event.Payload, &payload))
		assert.Equal(t, wantContent[index], payload["content"])
		assert.Equal(t, "user", payload["actor_role"])
		assert.NotContains(t, event.Payload, "系统自动回复不应进入通知")
		assert.NotContains(t, event.Payload, "管理员正文不应进入通知")
	}
}

func TestTicketNotificationContentPreservesUnicodeWithinEscapedBudget(t *testing.T) {
	const suffix = "\n[内容较长，已截断；完整内容请查看工单]"
	for _, tc := range []struct {
		name, content, want string
	}{
		{name: "原始纯文本", content: "<b>发票</b> & '🧾' \"内容\"", want: "<b>发票</b> & '🧾' \"内容\""},
		{name: "转义后恰好达到上限", content: strings.Repeat("\"", 400), want: strings.Repeat("\"", 400)},
		{name: "中文截断", content: strings.Repeat("文", 2001), want: strings.Repeat("文", 1979) + suffix},
		{name: "表情截断", content: strings.Repeat("🧾", 2001), want: strings.Repeat("🧾", 1979) + suffix},
		{name: "转义膨胀截断", content: "🧾中文" + strings.Repeat("\"", 401), want: "🧾中文" + strings.Repeat("\"", 395) + suffix},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupTicketTestDB(t)
			subscribeTicketEvents(t)
			ticket, err := CreateTicket(10, TicketCreateInput{Title: "正文长度", Category: "general", Priority: "normal", Content: tc.content})
			require.NoError(t, err)
			var event NotificationEvent
			require.NoError(t, DB.First(&event).Error)
			var payload map[string]any
			require.NoError(t, common.UnmarshalJsonStr(event.Payload, &payload))
			content, ok := payload["content"].(string)
			require.True(t, ok)
			assert.Equal(t, tc.want, content)
			assert.True(t, utf8.ValidString(content))
			assert.LessOrEqual(t, utf8.RuneCountInString(html.EscapeString(content)), 2000)
			detail, err := GetTicketDetail(ticket.Id, 10, false, 0)
			require.NoError(t, err)
			require.Len(t, detail.Messages, 1)
			assert.Equal(t, tc.content, detail.Messages[0].Content, "通知截断不能影响工单原文")
		})
	}
}

func TestTicketNotificationDefaultTemplateUpgradePreservesCustomTemplates(t *testing.T) {
	const oldDefault = "{{mention}} {{action_label}} #{{ticket_id}}：{{title}}\n用户：{{user_id}}\n分类：{{category}} 优先级：{{priority}}"
	const newDefault = oldDefault + "\n正文：{{content}}"
	for _, tc := range []struct {
		name, template, want string
	}{
		{name: "空模板", want: newDefault},
		{name: "旧默认模板", template: oldDefault, want: newDefault},
		{name: "新默认模板", template: newDefault, want: newDefault},
		{name: "自定义模板", template: "自定义：{{title}}", want: "自定义：{{title}}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, NormalizeNotificationTaskTemplate(NotificationEventTypeTicketUserMessage, tc.template))
		})
	}
	assert.Equal(t, oldDefault, NormalizeNotificationTaskTemplate(NotificationEventTypeChannelDisabled, oldDefault), "不能修改其他事件的自定义模板")

	setupTicketTestDB(t)
	tasks := []NotificationTask{
		{Name: "已合并旧默认", EventType: NotificationEventTypeTicketUserMessage, Template: oldDefault},
		{Name: "已合并自定义", EventType: NotificationEventTypeTicketUserMessage, Template: "自定义：{{title}}"},
		{Name: "已合并空模板", EventType: NotificationEventTypeTicketUserMessage},
		{Name: "已合并新默认", EventType: NotificationEventTypeTicketUserMessage, Template: newDefault},
		{Name: "旧追问默认", EventType: "ticket_user_replied", Template: "{{mention}} 工单 #{{ticket_id}} 收到用户追问：{{title}}\n用户：{{user_id}}"},
		{Name: "旧追问自定义", EventType: "ticket_user_replied", Template: "自定义：{{title}}"},
	}
	for index := range tasks {
		tasks[index].Enabled = true
		require.NoError(t, DB.Create(&tasks[index]).Error)
	}
	for round := 0; round < 2; round++ {
		require.NoError(t, migrateTicketNotificationTasks())
		var stored []NotificationTask
		require.NoError(t, DB.Order("id").Find(&stored).Error)
		require.Len(t, stored, len(tasks))
		for index, task := range stored {
			assert.Equal(t, NotificationEventTypeTicketUserMessage, task.EventType)
			assert.True(t, task.Enabled)
			if index == 1 || index == 5 {
				assert.Equal(t, "自定义：{{title}}", task.Template)
				continue
			}
			assert.Equal(t, newDefault, task.Template)
		}
	}
}
