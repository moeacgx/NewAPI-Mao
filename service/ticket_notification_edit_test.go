package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTicketNotificationEditTest(t *testing.T) *model.NotificationTask {
	t.Helper()
	setupNotificationServiceTestDB(t)
	model.InitDBColumns()
	t.Cleanup(model.InitDBColumns)
	require.NoError(t, model.DB.AutoMigrate(&model.Ticket{}, &model.TicketMessage{}, &model.Option{}, &model.TicketNotificationMessage{}))
	return subscribeTicketServiceNotifications(t, "")
}

func TestTicketMessageEditInterruptedFinalAttemptDoesNotRequestAgain(t *testing.T) {
	setupTicketNotificationEditTest(t)
	requests := make(chan telegramMessageRequest, 3)
	useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request telegramMessageRequest
		if err := common.DecodeJson(r.Body, &request); err != nil {
			w.WriteHeader(400)
			return
		}
		requests <- request
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":501}}`))
	}))
	ticket, err := model.CreateTicket(6494, model.TicketCreateInput{Title: "支付", Category: "billing", Priority: "normal", Content: "请协助"})
	require.NoError(t, err)
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	require.Len(t, requests, 1)
	<-requests
	_, err = model.ReplyTicket(ticket.Id, 1, true, "已处理")
	require.NoError(t, err)
	var tracking model.TicketNotificationMessage
	require.NoError(t, model.DB.First(&tracking).Error)
	require.NoError(t, model.DB.Model(&tracking).Updates(map[string]any{
		"status": model.NotificationDeliveryRetrying, "attempt_count": 8, "next_attempt_at": time.Now().Unix() - 1,
	}).Error)
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	require.NoError(t, model.DB.First(&tracking).Error)
	assert.Equal(t, model.NotificationDeliveryDead, tracking.Status)
	assert.Equal(t, 8, tracking.AttemptCount)
	assert.Empty(t, requests, "最后一次领取中断恢复后不能发起第九次请求")
}

func TestTicketMessageTrackingExpiresWithoutReply(t *testing.T) {
	setupTicketNotificationEditTest(t)
	now := time.Now().Unix()
	for index, status := range []string{model.NotificationDeliveryPending, model.NotificationDeliveryRetrying, model.NotificationDeliveryDead} {
		require.NoError(t, model.DB.Create(&model.TicketNotificationMessage{
			DeliveryId: index + 1, TicketId: 1, UserMessageId: 1, BotId: 1,
			ChatId: "-100", MessageId: index + 1, Content: "待清理原文", Status: status,
			CreatedAt: now - 3*24*60*60, NextAttemptAt: now + 3600,
		}).Error)
		require.NoError(t, model.DB.Create(&model.TicketNotificationMessage{
			DeliveryId: index + 4, TicketId: 1, UserMessageId: 1, BotId: 1,
			ChatId: "-100", MessageId: index + 4, Content: "未满三天原文", Status: status,
			CreatedAt: now - 71*60*60,
		}).Error)
	}
	require.True(t, (NotificationDispatcherSystemTaskHandler{}).Enabled())
	require.NoError(t, dispatchTicketNotificationEdits(context.Background()))
	var remaining []model.TicketNotificationMessage
	require.NoError(t, model.DB.Order("id").Find(&remaining).Error)
	require.Len(t, remaining, 3, "无回复和永久失败的关联满三天都应释放，未到期仍保留")
	for _, message := range remaining {
		assert.Equal(t, "未满三天原文", message.Content)
	}
	require.False(t, (NotificationDispatcherSystemTaskHandler{}).Enabled())
}

func TestTicketMessageTrackingClearsForUnusedRuleAfterHistoryPruning(t *testing.T) {
	for _, operation := range []string{"停用", "更新为停用", "物理删除"} {
		t.Run(operation, func(t *testing.T) {
			task := setupTicketNotificationEditTest(t)
			other := &model.NotificationTask{Name: "其他规则", EventType: task.EventType, BotId: task.BotId, Template: task.Template, Enabled: true}
			require.NoError(t, model.CreateNotificationTask(other))
			require.NoError(t, model.CreateNotificationTarget(&model.NotificationTarget{TaskId: other.Id, ChatId: "-10002", Enabled: true}))
			useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":701}}`))
			}))
			for index := 0; index < 6; index++ {
				_, err := model.CreateTicket(6494, model.TicketCreateInput{Title: fmt.Sprint(index), Category: "billing", Priority: "normal", Content: "请协助"})
				require.NoError(t, err)
				_, err = RunNotificationDispatcherPass(context.Background())
				require.NoError(t, err)
			}
			var before []model.TicketNotificationMessage
			require.NoError(t, model.DB.Order("id").Find(&before).Error)
			require.Len(t, before, 12)
			var deliveryCount int64
			require.NoError(t, model.DB.Model(&model.NotificationDelivery{}).Where("task_id = ?", task.Id).Count(&deliveryCount).Error)
			require.EqualValues(t, 5, deliveryCount, "已经清理了第一条投递历史")
			switch operation {
			case "停用":
				require.NoError(t, model.SetNotificationTaskEnabled(task.Id, false))
			case "更新为停用":
				task.Enabled = false
				require.NoError(t, model.UpdateNotificationTask(task))
			case "物理删除":
				require.NoError(t, model.DeleteNotificationTask(task.Id))
			}
			var remaining []model.TicketNotificationMessage
			require.NoError(t, model.DB.Order("id").Find(&remaining).Error)
			require.Len(t, remaining, 6, "规则清理必须覆盖已被裁剪投递历史的消息")
			for _, message := range remaining {
				assert.Equal(t, other.Id, message.TaskId, "不能误清其他规则，即使使用相同 Bot")
			}
		})
	}
}

func TestTicketMessageTrackingNotRecreatedWhenRuleDisabledDuringSend(t *testing.T) {
	for _, reenable := range []bool{false, true} {
		t.Run(fmt.Sprintf("立即重新启用=%t", reenable), func(t *testing.T) {
			task := setupTicketNotificationEditTest(t)
			firstSend := true
			errors := make(chan error, 2)
			requests := make(chan telegramMessageRequest, 3)
			useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request telegramMessageRequest
				if err := common.DecodeJson(r.Body, &request); err != nil {
					w.WriteHeader(400)
					return
				}
				requests <- request
				if firstSend {
					firstSend = false
					errors <- model.SetNotificationTaskEnabled(task.Id, false)
					if reenable {
						errors <- model.SetNotificationTaskEnabled(task.Id, true)
					}
				}
				_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":801}}`))
			}))
			ticket, err := model.CreateTicket(6494, model.TicketCreateInput{Title: "支付", Category: "billing", Priority: "normal", Content: "请协助"})
			require.NoError(t, err)
			_, err = RunNotificationDispatcherPass(context.Background())
			require.NoError(t, err)
			expectedOperations := 1
			if reenable {
				expectedOperations = 2
			}
			require.Len(t, errors, expectedOperations)
			for len(errors) > 0 {
				require.NoError(t, <-errors)
			}
			var count int64
			require.NoError(t, model.DB.Model(&model.TicketNotificationMessage{}).Count(&count).Error)
			assert.Zero(t, count, "停用期间已经发出的旧通知不能重新保存跟踪")
			var delivery model.NotificationDelivery
			require.NoError(t, model.DB.First(&delivery).Error)
			assert.Equal(t, model.NotificationDeliverySuccess, delivery.Status, "已经发送成功不能误记失败或自动重发")
			require.Len(t, requests, 1)
			<-requests
			if reenable {
				_, err = model.ReplyTicket(ticket.Id, 6494, false, "新的追问")
				require.NoError(t, err)
				_, err = RunNotificationDispatcherPass(context.Background())
				require.NoError(t, err)
				require.Len(t, requests, 1)
				assert.Zero(t, (<-requests).MessageID)
				require.NoError(t, model.DB.Model(&model.TicketNotificationMessage{}).Count(&count).Error)
				assert.EqualValues(t, 1, count, "重新启用后新事件正常保存跟踪")
			}
		})
	}
}

func TestTicketNotificationReservesExactReplyMarkerBudget(t *testing.T) {
	for _, length := range []int{4090, 4091} {
		t.Run(fmt.Sprint(length), func(t *testing.T) {
			task := setupTicketNotificationEditTest(t)
			require.NoError(t, model.DB.Model(task).Update("template", strings.Repeat("字", length)).Error)
			requests := make(chan telegramMessageRequest, 3)
			useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request telegramMessageRequest
				if err := common.DecodeJson(r.Body, &request); err != nil {
					w.WriteHeader(400)
					return
				}
				requests <- request
				_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":601}}`))
			}))
			ticket, err := model.CreateTicket(6494, model.TicketCreateInput{Title: "支付", Category: "billing", Priority: "normal", Content: "请协助"})
			require.NoError(t, err)
			_, err = RunNotificationDispatcherPass(context.Background())
			require.NoError(t, err)
			if length == 4091 {
				assert.Empty(t, requests, "超过预留预算的通知在发送前拒绝")
				var delivery model.NotificationDelivery
				require.NoError(t, model.DB.First(&delivery).Error)
				assert.Equal(t, model.NotificationDeliveryDead, delivery.Status)
				return
			}
			require.Len(t, requests, 1)
			<-requests
			_, err = model.ReplyTicket(ticket.Id, 1, true, "已处理")
			require.NoError(t, err)
			_, err = RunNotificationDispatcherPass(context.Background())
			require.NoError(t, err)
			require.Len(t, requests, 1)
			assert.Equal(t, 4096, utf8.RuneCountInString((<-requests).Text))
			var count int64
			require.NoError(t, model.DB.Model(&model.TicketNotificationMessage{}).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}

func TestTicketNotificationHumanReplyEditsOriginalAndClearsTracking(t *testing.T) {
	setupTicketNotificationEditTest(t)
	require.NoError(t, model.DB.Create(&model.Option{Key: model.TicketAutoReplyOptionKey, Value: `{"enabled":true,"content":"自动确认"}`}).Error)
	requests := make(chan telegramMessageRequest, 10)
	useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request telegramMessageRequest
		if err := common.DecodeJson(r.Body, &request); err != nil {
			w.WriteHeader(400)
			return
		}
		requests <- request
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":101,"chat":{"id":-9000}}}`))
	}))
	ticket, err := model.CreateTicket(6494, model.TicketCreateInput{Title: "付款失败", Category: "billing", Priority: "normal", Content: "错误 <b>原文</b>"})
	require.NoError(t, err)
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	first := <-requests
	assert.Equal(t, 0, first.MessageID)
	assert.Contains(t, first.Text, "<blockquote>错误 &lt;b&gt;原文&lt;/b&gt;</blockquote>")
	assert.Empty(t, requests, "自动确认不能触发编辑")
	require.False(t, (NotificationDispatcherSystemTaskHandler{}).Enabled())
	_, err = model.ChangeTicketStatus(ticket.Id, 1, true, model.TicketStatusResolved)
	require.NoError(t, err)
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	assert.Empty(t, requests, "单独修改状态不能代替人工回复")
	_, err = model.ReplyTicket(ticket.Id, 1, true, "请使用另一种支付方式")
	require.NoError(t, err)
	require.True(t, (NotificationDispatcherSystemTaskHandler{}).Enabled())
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	edited := <-requests
	assert.Equal(t, 101, edited.MessageID)
	assert.Equal(t, "-9000", edited.ChatID, "保存 Telegram 实际数字群号")
	assert.Equal(t, "✅ 已回复\n"+first.Text, edited.Text)
	var remaining int64
	require.NoError(t, model.DB.Model(&model.TicketNotificationMessage{}).Count(&remaining).Error)
	assert.Zero(t, remaining, "成功编辑后立即清理关联和原文")
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	assert.Empty(t, requests)
	_, err = model.ReplyTicket(ticket.Id, 6494, false, "仍有问题")
	require.NoError(t, err)
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	assert.NotContains(t, (<-requests).Text, "✅ 已回复")
	assert.Empty(t, requests, "后续追问不能使用之前的人工回复提前打勾")
}

func TestDelayedTicketNotificationStillGetsReplyMarker(t *testing.T) {
	for _, duringSend := range []bool{false, true} {
		t.Run(fmt.Sprintf("发送中回复=%t", duringSend), func(t *testing.T) {
			setupTicketNotificationEditTest(t)
			ticket, err := model.CreateTicket(6494, model.TicketCreateInput{Title: "付款失败", Category: "billing", Priority: "normal", Content: "请协助"})
			require.NoError(t, err)
			if !duringSend {
				_, err = model.ReplyTicket(ticket.Id, 1, true, "已处理")
				require.NoError(t, err)
			}
			requests := make(chan telegramMessageRequest, 3)
			replyErrors := make(chan error, 1)
			useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request telegramMessageRequest
				if err := common.DecodeJson(r.Body, &request); err != nil {
					w.WriteHeader(400)
					return
				}
				requests <- request
				if request.MessageID == 0 && duringSend {
					_, err := model.ReplyTicket(ticket.Id, 1, true, "发送期间已回复")
					replyErrors <- err
				}
				_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":201,"chat":{"id":-9000}}}`))
			}))
			_, err = RunNotificationDispatcherPass(context.Background())
			require.NoError(t, err)
			if duringSend {
				require.NoError(t, <-replyErrors)
			}
			original, edited := <-requests, <-requests
			assert.Equal(t, 201, edited.MessageID)
			assert.Equal(t, "✅ 已回复\n"+original.Text, edited.Text)
		})
	}
}

func TestTicketReplyEditsSurviveHistoryPruningAndConfigurationChanges(t *testing.T) {
	task := setupTicketNotificationEditTest(t)
	requests := make(chan telegramMessageRequest, 10)
	nextID := 300
	useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request telegramMessageRequest
		if err := common.DecodeJson(r.Body, &request); err != nil {
			w.WriteHeader(400)
			return
		}
		requests <- request
		nextID++
		_, _ = fmt.Fprintf(w, `{"ok":true,"result":{"message_id":%d,"chat":{"id":-9000}}}`, nextID)
	}))
	var first *model.Ticket
	for index := 0; index < 6; index++ {
		ticket, err := model.CreateTicket(6494, model.TicketCreateInput{Title: fmt.Sprintf("问题%d", index), Category: "billing", Priority: "normal", Content: "请协助"})
		require.NoError(t, err)
		if index == 0 {
			first = ticket
		}
		_, err = RunNotificationDispatcherPass(context.Background())
		require.NoError(t, err)
	}
	original := <-requests
	for len(requests) > 0 {
		<-requests
	}
	var history []model.NotificationDelivery
	require.NoError(t, model.DB.Find(&history).Error)
	require.Len(t, history, 5)
	assert.Greater(t, history[0].Id, 1)
	require.NoError(t, model.DB.Model(&model.NotificationTarget{}).Where("task_id = ?", task.Id).Update("chat_id", "-8000").Error)
	require.NoError(t, model.DB.Model(task).Update("template", "模板已经改动 {{title}}").Error)
	_, err := model.ReplyTicket(first.Id, 1, true, "已解决")
	require.NoError(t, err)
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	edited := <-requests
	assert.Equal(t, 301, edited.MessageID)
	assert.Equal(t, "-9000", edited.ChatID)
	assert.Equal(t, "✅ 已回复\n"+original.Text, edited.Text)
}

func TestTicketMessageEditRetriesPerTargetAndAlreadyModifiedIsSuccess(t *testing.T) {
	task := setupTicketNotificationEditTest(t)
	require.NoError(t, model.CreateNotificationTarget(&model.NotificationTarget{TaskId: task.Id, ChatId: "-10002", Enabled: true}))
	requests := make(chan telegramMessageRequest, 10)
	limited := false
	useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request telegramMessageRequest
		if err := common.DecodeJson(r.Body, &request); err != nil {
			w.WriteHeader(400)
			return
		}
		requests <- request
		if request.MessageID > 0 && request.ChatID == "-10002" {
			if !limited {
				limited = true
				w.WriteHeader(429)
				_, _ = w.Write([]byte(`{"ok":false,"parameters":{"retry_after":30}}`))
				return
			}
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: message is not modified: specified new message content is exactly the same"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":401}}`))
	}))
	ticket, err := model.CreateTicket(6494, model.TicketCreateInput{Title: "支付", Category: "billing", Priority: "normal", Content: "请协助"})
	require.NoError(t, err)
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	_, err = model.ReplyTicket(ticket.Id, 1, true, "已处理")
	require.NoError(t, err)
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	var remaining []model.TicketNotificationMessage
	require.NoError(t, model.DB.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, "-10002", remaining[0].ChatId)
	assert.Equal(t, model.NotificationDeliveryRetrying, remaining[0].Status)
	assert.Equal(t, 1, remaining[0].AttemptCount)
	assert.Greater(t, remaining[0].NextAttemptAt, time.Now().Unix())
	assert.NotContains(t, remaining[0].LastError, "test-token")
	require.NoError(t, model.DB.Model(&remaining[0]).Update("next_attempt_at", 0).Error)
	_, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	var count int64
	require.NoError(t, model.DB.Model(&model.TicketNotificationMessage{}).Count(&count).Error)
	assert.Zero(t, count)
	sends, edits := 0, 0
	for len(requests) > 0 {
		request := <-requests
		if request.MessageID == 0 {
			sends++
		} else {
			edits++
			assert.True(t, strings.HasPrefix(request.Text, "✅ 已回复\n"))
		}
	}
	assert.Equal(t, 2, sends)
	assert.Equal(t, 3, edits)
}
