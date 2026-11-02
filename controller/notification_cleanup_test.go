package controller

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDisableNotificationRuleCleansTrackingAtomically(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupNotificationControllerTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.TicketNotificationMessage{}))
	bot := &model.NotificationBot{Name: "工单机器人", Token: "test-token", Enabled: true}
	require.NoError(t, model.CreateNotificationBot(bot))
	task := &model.NotificationTask{Name: "待停用规则", EventType: model.NotificationEventTypeTicketUserMessage, BotId: bot.Id, Enabled: true}
	require.NoError(t, model.CreateNotificationTask(task))
	require.NoError(t, model.CreateNotificationTarget(&model.NotificationTarget{TaskId: task.Id, ChatId: "-100", Enabled: true}))
	require.NoError(t, model.EnqueueNotificationEvent(task.EventType, "cleanup-test", map[string]any{"ticket_id": 1}))
	tracking := model.TicketNotificationMessage{
		DeliveryId: 999, TaskId: task.Id, TicketId: 1, UserMessageId: 1, BotId: bot.Id,
		ChatId: "-100", MessageId: 701, Content: "旧原文", Status: model.NotificationDeliveryPending, CreatedAt: time.Now().Unix(),
	}
	require.NoError(t, model.DB.Create(&tracking).Error)
	cleanupErr := errors.New("模拟关联清理失败")
	require.NoError(t, model.DB.Callback().Delete().Before("gorm:delete").Register("test:reject-tracking-cleanup", func(tx *gorm.DB) {
		if tx.Statement.Table == "ticket_notification_messages" {
			tx.AddError(cleanupErr)
		}
	}))
	path := fmt.Sprintf("/api/notification/tasks/%d", task.Id)
	failed := invokeNotificationHandler(DisableNotificationTask, common.RoleRootUser, http.MethodDelete, path, "", task.Id)
	assert.Contains(t, failed.Body.String(), `"success":false`)
	require.NoError(t, model.DB.First(task, task.Id).Error)
	assert.True(t, task.Enabled, "清理失败时必须回滚规则停用")
	var delivery model.NotificationDelivery
	require.NoError(t, model.DB.First(&delivery).Error)
	assert.Equal(t, model.NotificationDeliveryPending, delivery.Status, "取消待发送投递也必须回滚")
	var count int64
	require.NoError(t, model.DB.Model(&model.TicketNotificationMessage{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
	require.NoError(t, model.DB.Callback().Delete().Remove("test:reject-tracking-cleanup"))

	succeeded := invokeNotificationHandler(DisableNotificationTask, common.RoleRootUser, http.MethodDelete, path, "", task.Id)
	assert.Contains(t, succeeded.Body.String(), `"success":true`)
	require.NoError(t, model.DB.First(task, task.Id).Error)
	assert.False(t, task.Enabled)
	require.NoError(t, model.DB.First(&delivery, delivery.Id).Error)
	assert.Equal(t, model.NotificationDeliveryCanceled, delivery.Status)
	require.NoError(t, model.DB.Model(&model.TicketNotificationMessage{}).Count(&count).Error)
	assert.Zero(t, count, "实际 DELETE 接口成功后不残留消息 ID 或原文")
}
