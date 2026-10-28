package model

import (
	"html"
	"unicode/utf8"

	"gorm.io/gorm"
)

// 工单只通知管理员需要处理的用户动作：新建工单与用户追加回复合并为一个事件。
// 管理员回复、状态变更（含批量改状态和自动归档）由管理员自己发起，不再外发通知。
const (
	NotificationEventTypeTicketUserMessage = "ticket_user_message"
	NotificationTicketUserMessageTemplate  = legacyNotificationTicketUserMessageTemplate + "\n正文：{{content}}"

	// TicketNotificationContentMaxEscapedRunes 为正文快照预留统一 HTML 转义后的字符预算。
	TicketNotificationContentMaxEscapedRunes = 2000

	legacyNotificationTicketUserMessageTemplate = "{{mention}} {{action_label}} #{{ticket_id}}：{{title}}\n用户：{{user_id}}\n分类：{{category}} 优先级：{{priority}}"
	ticketNotificationContentTruncatedSuffix    = "\n[内容较长，已截断；完整内容请查看工单]"

	TicketNotificationActionCreated = "created"
	TicketNotificationActionReplied = "replied"
)

// 已合并或下线的旧事件值，仅用于迁移历史通知任务。
var (
	legacyTicketUserNotificationEventTypes  = []string{"ticket_created", "ticket_user_replied"}
	legacyTicketAdminNotificationEventTypes = []string{"ticket_admin_replied", "ticket_status_changed"}
	legacyTicketUserNotificationTemplates   = []string{
		"",
		NotificationTaskDefaultTemplate,
		legacyNotificationTicketUserMessageTemplate,
		"{{mention}} 新工单 #{{ticket_id}}：{{title}}\n用户：{{user_id}}\n分类：{{category}} 优先级：{{priority}}",
		"{{mention}} 工单 #{{ticket_id}} 收到用户追问：{{title}}\n用户：{{user_id}}",
	}
)

var ticketNotificationActionLabels = map[string]string{
	TicketNotificationActionCreated: "新工单",
	TicketNotificationActionReplied: "用户追加回复",
}

// ticketNotificationContentSnapshot 保存当次用户正文，不重查后续消息，也不预转义。
// 按原文字符边界截断，并将提示计入 HTML 转义后的预算，供统一渲染器安全投递。
func ticketNotificationContentSnapshot(content string) string {
	if utf8.RuneCountInString(html.EscapeString(content)) <= TicketNotificationContentMaxEscapedRunes {
		return content
	}
	budget := TicketNotificationContentMaxEscapedRunes - utf8.RuneCountInString(html.EscapeString(ticketNotificationContentTruncatedSuffix))
	escapedRunes := 0
	for index, character := range content {
		escapedRunes += utf8.RuneCountInString(html.EscapeString(string(character)))
		if escapedRunes > budget {
			return content[:index] + ticketNotificationContentTruncatedSuffix
		}
	}
	return content
}

// migrateTicketNotificationTasks 把旧的新建/用户追问任务并入合并事件，并停用已下线事件的任务。
// 已合并任务和旧事件任务中的旧默认模板升级为含正文的模板，自定义模板保留；可重复执行。
// 同时订阅了新建与追问的部署迁移后会各收到一份，需在通知中心删除重复任务。
func migrateTicketNotificationTasks() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		userEventTypes := append([]string{NotificationEventTypeTicketUserMessage}, legacyTicketUserNotificationEventTypes...)
		if err := tx.Model(&NotificationTask{}).
			Where("event_type IN ? AND template IN ?", userEventTypes, legacyTicketUserNotificationTemplates).
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
