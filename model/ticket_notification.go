package model

import "gorm.io/gorm"

// 工单只通知管理员需要处理的用户动作：新建工单与用户追加回复合并为一个事件。
// 管理员回复、状态变更（含批量改状态和自动归档）由管理员自己发起，不再外发通知。
const (
	NotificationEventTypeTicketUserMessage = "ticket_user_message"
	NotificationTicketUserMessageTemplate  = "{{mention}} {{action_label}} #{{ticket_id}}：{{title}}\n用户：{{user_id}}\n分类：{{category}} 优先级：{{priority}}"

	TicketNotificationActionCreated = "created"
	TicketNotificationActionReplied = "replied"
)

// 已合并或下线的旧事件值，仅用于迁移历史通知任务。
var (
	legacyTicketUserNotificationEventTypes  = []string{"ticket_created", "ticket_user_replied"}
	legacyTicketAdminNotificationEventTypes = []string{"ticket_admin_replied", "ticket_status_changed"}
	legacyTicketUserNotificationTemplates   = []string{
		"",
		"{{mention}} 新工单 #{{ticket_id}}：{{title}}\n用户：{{user_id}}\n分类：{{category}} 优先级：{{priority}}",
		"{{mention}} 工单 #{{ticket_id}} 收到用户追问：{{title}}\n用户：{{user_id}}",
	}
)

var ticketNotificationActionLabels = map[string]string{
	TicketNotificationActionCreated: "新工单",
	TicketNotificationActionReplied: "用户追加回复",
}

// migrateTicketNotificationTasks 把旧的新建/用户追问任务并入合并事件，并停用已下线事件的任务。
// 仍为旧默认模板的任务换成合并后的默认模板，自定义模板保留；可重复执行。
// 同时订阅了新建与追问的部署迁移后会各收到一份，需在通知中心删除重复任务。
func migrateTicketNotificationTasks() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&NotificationTask{}).
			Where("event_type IN ? AND template IN ?", legacyTicketUserNotificationEventTypes, legacyTicketUserNotificationTemplates).
			Update("template", NotificationTicketUserMessageTemplate).Error; err != nil {
			return err
		}
		if err := tx.Model(&NotificationTask{}).
			Where("event_type IN ?", legacyTicketUserNotificationEventTypes).
			Update("event_type", NotificationEventTypeTicketUserMessage).Error; err != nil {
			return err
		}
		return tx.Model(&NotificationTask{}).
			Where("event_type IN ? AND enabled = ?", legacyTicketAdminNotificationEventTypes, true).
			Update("enabled", false).Error
	})
}
