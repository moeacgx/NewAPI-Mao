package service

import (
	"context"
	"html"
	"net/http"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func subscribeTicketServiceNotifications(t *testing.T, template string) *model.NotificationTask {
	t.Helper()
	bot := &model.NotificationBot{Name: "工单测试", Token: "test-token", Enabled: true}
	require.NoError(t, model.CreateNotificationBot(bot))
	task := &model.NotificationTask{Name: "工单正文", EventType: model.NotificationEventTypeTicketUserMessage, BotId: bot.Id, Template: template, Enabled: true}
	require.NoError(t, model.CreateNotificationTask(task))
	require.NoError(t, model.CreateNotificationTarget(&model.NotificationTarget{
		TaskId: task.Id, ChatId: "-10001", Enabled: true,
		MentionUserId: "9223372036854775807", MentionName: strings.Repeat("\"", 128),
	}))
	return task
}

func TestDispatchTicketNotificationSendsEscapedUserContentWithinLimit(t *testing.T) {
	for _, tc := range []struct {
		name, content, want string
		truncated           bool
	}{
		{"普通正文", "请开票 <b>不是标签</b> & 保留换行\n第二行", "正文：请开票 &lt;b&gt;不是标签&lt;/b&gt; &amp; 保留换行\n第二行", false},
		{"超长中文和表情", strings.Repeat("开票😀", 3000), "正文：开票😀", true},
		{"转义膨胀", strings.Repeat("\"<&'😀", 2000), "正文：&#34;&lt;&amp;&#39;😀", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupNotificationServiceTestDB(t)
			model.InitDBColumns()
			t.Cleanup(model.InitDBColumns)
			require.NoError(t, model.DB.AutoMigrate(&model.Ticket{}, &model.TicketMessage{}, &model.Option{}))
			subscribeTicketServiceNotifications(t, "")
			ticket, err := model.CreateTicket(8257, model.TicketCreateInput{
				Title: strings.Repeat("\"", model.TicketTitleMaxRunes), Category: "billing", Priority: "normal", Content: tc.content,
			})
			require.NoError(t, err)
			var stored model.TicketMessage
			require.NoError(t, model.DB.Where("ticket_id = ?", ticket.Id).First(&stored).Error)
			assert.Equal(t, tc.content, stored.Content, "通知摘要不得截断工单原文")
			requests := make(chan telegramMessageRequest, 1)
			useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request telegramMessageRequest
				if err := common.DecodeJson(r.Body, &request); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				requests <- request
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			result, err := RunNotificationDispatcherPass(context.Background())
			require.NoError(t, err)
			require.Equal(t, 1, result.Processed)
			var delivery model.NotificationDelivery
			require.NoError(t, model.DB.First(&delivery).Error)
			require.Equal(t, model.NotificationDeliverySuccess, delivery.Status, delivery.LastError)
			select {
			case request := <-requests:
				assert.Equal(t, "-10001", request.ChatID)
				assert.Equal(t, "HTML", request.ParseMode)
				assert.Contains(t, request.Text, tc.want)
				assert.LessOrEqual(t, utf8.RuneCountInString(request.Text), 4096)
				assert.True(t, utf8.ValidString(request.Text))
				assert.NotContains(t, request.Text, "<b>不是标签</b>")
				if tc.truncated {
					assert.True(t, strings.HasSuffix(request.Text, "\n[内容较长，已截断；完整内容请查看工单]"))
					_, body, found := strings.Cut(html.UnescapeString(request.Text), "正文：")
					require.True(t, found, "默认模板应包含正文")
					assert.True(t, strings.HasPrefix(tc.content, strings.Split(body, "\n[内容较长")[0]))
				}
			default:
				t.Fatal("工单通知未发出 HTTP 请求")
			}
		})
	}
}

func TestDispatchHistoricalTicketNotificationUsesPlaceholderAndKeepsUnknownVariableErrors(t *testing.T) {
	const legacyTemplate = "{{mention}} {{action_label}} #{{ticket_id}}：{{title}}\n用户：{{user_id}}\n分类：{{category}} 优先级：{{priority}}"
	for _, tc := range []struct {
		name, template string
		valid          bool
	}{
		{"旧默认模板", legacyTemplate, true},
		{"自定义正文模板", "{{content}}", true},
		{"其他未知变量仍拒绝", "{{content}} {{unknown}}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupNotificationServiceTestDB(t)
			task := subscribeTicketServiceNotifications(t, "{{title}}")
			require.NoError(t, model.DB.Model(task).Update("template", tc.template).Error)
			require.NoError(t, model.DB.Transaction(func(tx *gorm.DB) error {
				return model.EnqueueNotificationEventTx(tx, model.NotificationEventTypeTicketUserMessage, "historical-ticket", map[string]any{
					"ticket_id": 1, "title": "开发票问题", "user_id": 8257, "category": "billing", "priority": "normal", "action_label": "用户追加回复",
				})
			}))
			requests := make(chan telegramMessageRequest, 1)
			useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request telegramMessageRequest
				if err := common.DecodeJson(r.Body, &request); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				requests <- request
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			work, err := model.ClaimNotificationDeliveries(1, time.Now().Unix(), 120)
			require.NoError(t, err)
			require.Len(t, work, 1)
			dispatchNotificationDelivery(context.Background(), work[0])
			var delivery model.NotificationDelivery
			require.NoError(t, model.DB.First(&delivery).Error)
			if !tc.valid {
				assert.Equal(t, model.NotificationDeliveryDead, delivery.Status)
				assert.Contains(t, delivery.LastError, "unknown notification template variable: unknown")
				assert.Empty(t, requests)
				return
			}
			require.Equal(t, model.NotificationDeliverySuccess, delivery.Status, delivery.LastError)
			select {
			case request := <-requests:
				assert.Contains(t, request.Text, "历史通知未保存正文，请查看工单")
			default:
				t.Fatal("历史工单通知未发出 HTTP 请求")
			}
			var event model.NotificationEvent
			require.NoError(t, model.DB.First(&event).Error)
			assert.NotContains(t, event.Payload, `"content"`, "历史负载不得被回写")
		})
	}
}
