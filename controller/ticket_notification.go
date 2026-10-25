package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
)

// 工单只提供一个通知事件：用户新建工单或追加回复。action 为 created/replied，
// action_label 为对应的中文说明，便于一个模板同时覆盖两种情况。
func ticketNotificationEventDefinitions() []notificationEventTypeDefinition {
	return []notificationEventTypeDefinition{{
		Value:           model.NotificationEventTypeTicketUserMessage,
		Label:           "Ticket created or user replied",
		Description:     "Triggered when a user creates a support ticket or adds a reply.",
		Owner:           "core",
		Available:       true,
		DefaultTemplate: model.NormalizeNotificationTaskTemplate(model.NotificationEventTypeTicketUserMessage, ""),
		Variables:       []string{"mention", "ticket_id", "title", "user_id", "category", "priority", "status", "actor_id", "actor_role", "create_time", "action", "action_label"},
		SamplePayload: map[string]any{
			"ticket_id": 123, "title": strings.Repeat("工", model.TicketTitleMaxRunes), "user_id": 42,
			"category": "technical", "priority": "normal", "status": "open", "actor_id": 42, "actor_role": "user",
			"create_time": 1792627200, "action": model.TicketNotificationActionReplied, "action_label": "用户追加回复",
		},
	}}
}
