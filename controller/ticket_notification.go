package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
)

func ticketNotificationEventDefinitions() []notificationEventTypeDefinition {
	events := []struct{ value, label, description string }{
		{model.NotificationEventTypeTicketCreated, "Ticket created", "Triggered when a user creates a support ticket."},
		{model.NotificationEventTypeTicketUserReplied, "Ticket user replied", "Triggered when a user replies to a support ticket."},
		{model.NotificationEventTypeTicketAdminReplied, "Ticket administrator replied", "Triggered when an administrator replies to a support ticket."},
		{model.NotificationEventTypeTicketStatusChanged, "Ticket status changed", "Triggered when a support ticket status is changed."},
	}
	definitions := make([]notificationEventTypeDefinition, 0, len(events))
	for _, event := range events {
		definitions = append(definitions, notificationEventTypeDefinition{
			Value: event.value, Label: event.label, Description: event.description,
			Owner: "core", Available: true,
			DefaultTemplate: model.NormalizeNotificationTaskTemplate(event.value, ""),
			Variables:       []string{"mention", "ticket_id", "title", "user_id", "category", "priority", "status", "actor_id", "actor_role", "create_time"},
			SamplePayload:   map[string]any{"ticket_id": 123, "title": strings.Repeat("工", model.TicketTitleMaxRunes), "user_id": 42, "category": "technical", "priority": "normal", "status": "processing", "actor_id": 42, "actor_role": "admin", "create_time": 1792627200},
		})
	}
	return definitions
}
