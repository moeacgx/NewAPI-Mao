package authz

const ResourceSupplierApplications = "supplier_applications"

var (
	SupplierApplicationAdminRead  = Permission{Resource: ResourceSupplierApplications, Action: ActionRead}
	SupplierApplicationAdminWrite = Permission{Resource: ResourceSupplierApplications, Action: ActionWrite}
)

func init() {
	RegisterResource(ResourceDefinition{
		Resource: ResourceSupplierApplications,
		LabelKey: "Supplier Application Management",
		Actions: []ActionDefinition{
			{Action: ActionRead, LabelKey: "Read supplier applications", DescriptionKey: "View all supplier applications, contact details, group quotes, and internal notes.", DefaultRoles: []string{BuiltInRoleAdmin}},
			{Action: ActionWrite, LabelKey: "Process supplier applications", DescriptionKey: "Update supplier application status and internal notes, and record conversation messages.", DefaultRoles: []string{BuiltInRoleAdmin}},
		},
	})
}
