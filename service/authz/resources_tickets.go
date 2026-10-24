package authz

const ResourceTickets = "tickets"

var (
	TicketAdminRead  = Permission{Resource: ResourceTickets, Action: ActionRead}
	TicketAdminWrite = Permission{Resource: ResourceTickets, Action: ActionWrite}
)

func init() {
	RegisterResource(ResourceDefinition{
		Resource: ResourceTickets,
		LabelKey: "Ticket Management",
		Actions: []ActionDefinition{
			{Action: ActionRead, LabelKey: "Read all tickets", DescriptionKey: "View all support tickets and messages.", DefaultRoles: []string{BuiltInRoleAdmin}},
			{Action: ActionWrite, LabelKey: "Manage tickets", DescriptionKey: "Reply to support tickets, update their status, delete tickets, and manage automation.", DefaultRoles: []string{BuiltInRoleAdmin}},
		},
	})
}
