package auth

// Permission constants follow the "resource.action" convention.
const (
	PermLeadsRead      Permission = "leads.read"
	PermLeadsCreate    Permission = "leads.create"
	PermLeadsUpdate    Permission = "leads.update"
	PermLeadsDelete    Permission = "leads.delete"
	PermApplicantsRead   Permission = "applicants.read"
	PermApplicantsCreate Permission = "applicants.create"
	PermApplicantsUpdate Permission = "applicants.update"
	PermApplicantsDelete Permission = "applicants.delete"
	PermVisaCasesRead    Permission = "visa_cases.read"
	PermVisaCasesCreate  Permission = "visa_cases.create"
	PermVisaCasesUpdate  Permission = "visa_cases.update"
	PermDocumentsRead   Permission = "documents.read"
	PermDocumentsUpload Permission = "documents.upload"
	PermDocumentsUpdate Permission = "documents.update"
	PermDocumentsDelete Permission = "documents.delete"
	PermPaymentsRead   Permission = "payments.read"
	PermPaymentsCreate Permission = "payments.create"
	PermPaymentsUpdate Permission = "payments.update"
	PermReportsRead    Permission = "reports.read"
)

func perms(list ...Permission) map[Permission]struct{} {
	set := make(map[Permission]struct{}, len(list))
	for _, p := range list {
		set[p] = struct{}{}
	}
	return set
}

var rolePermissions = map[Role]map[Permission]struct{}{
	RoleSuperAdmin: perms(
		PermLeadsRead, PermLeadsCreate, PermLeadsUpdate, PermLeadsDelete,
		PermApplicantsRead, PermApplicantsCreate, PermApplicantsUpdate, PermApplicantsDelete,
		PermVisaCasesRead, PermVisaCasesCreate, PermVisaCasesUpdate,
		PermDocumentsRead, PermDocumentsUpload, PermDocumentsUpdate, PermDocumentsDelete,
		PermPaymentsRead, PermPaymentsCreate, PermPaymentsUpdate,
		PermReportsRead,
	),
	RoleAdmin: perms(
		PermLeadsRead, PermLeadsCreate, PermLeadsUpdate, PermLeadsDelete,
		PermApplicantsRead, PermApplicantsCreate, PermApplicantsUpdate, PermApplicantsDelete,
		PermVisaCasesRead, PermVisaCasesCreate, PermVisaCasesUpdate,
		PermDocumentsRead, PermDocumentsUpload, PermDocumentsUpdate, PermDocumentsDelete,
		PermPaymentsRead, PermPaymentsCreate, PermPaymentsUpdate,
		PermReportsRead,
	),
	RoleManager: perms(
		PermLeadsRead, PermLeadsCreate, PermLeadsUpdate,
		PermApplicantsRead, PermApplicantsCreate, PermApplicantsUpdate,
		PermVisaCasesRead, PermVisaCasesCreate, PermVisaCasesUpdate,
		PermDocumentsRead, PermDocumentsUpload, PermDocumentsUpdate,
		PermPaymentsRead,
		PermReportsRead,
	),
	RoleSales: perms(
		PermLeadsRead, PermLeadsCreate, PermLeadsUpdate,
		PermApplicantsRead, PermApplicantsCreate, PermApplicantsUpdate,
		PermVisaCasesRead, PermVisaCasesCreate, PermVisaCasesUpdate,
		PermDocumentsRead, PermDocumentsUpload,
	),
	RoleStaff: perms(
		PermLeadsRead,
		PermApplicantsRead,
		PermVisaCasesRead,
		PermDocumentsRead, PermDocumentsUpload,
	),
	RoleAccountant: perms(
		PermLeadsRead,
		PermApplicantsRead,
		PermVisaCasesRead,
		PermDocumentsRead,
		PermPaymentsRead, PermPaymentsCreate, PermPaymentsUpdate,
		PermReportsRead,
	),
}

func roleHasPermission(role Role, p Permission) bool {
	set, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, allowed := set[p]
	return allowed
}
