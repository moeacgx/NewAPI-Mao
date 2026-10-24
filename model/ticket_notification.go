package model

const (
	NotificationEventTypeTicketCreated       = "ticket_created"
	NotificationEventTypeTicketUserReplied   = "ticket_user_replied"
	NotificationEventTypeTicketAdminReplied  = "ticket_admin_replied"
	NotificationEventTypeTicketStatusChanged = "ticket_status_changed"
	NotificationTicketCreatedTemplate        = "{{mention}} 新工单 #{{ticket_id}}：{{title}}\n用户：{{user_id}}\n分类：{{category}} 优先级：{{priority}}"
	NotificationTicketUserRepliedTemplate    = "{{mention}} 工单 #{{ticket_id}} 收到用户追问：{{title}}\n用户：{{user_id}}"
	NotificationTicketAdminRepliedTemplate   = "{{mention}} 管理员已回复工单 #{{ticket_id}}：{{title}}\n用户：{{user_id}}"
	NotificationTicketStatusChangedTemplate  = "{{mention}} 工单 #{{ticket_id}} 状态更新：{{status}}\n标题：{{title}}\n操作人：{{actor_id}}（{{actor_role}}）"
)
