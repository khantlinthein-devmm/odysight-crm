package auth

// Permission constants follow the "resource.action" convention.
const (
	PermLeadsRead            Permission = "leads.read"
	PermLeadsCreate          Permission = "leads.create"
	PermLeadsUpdate          Permission = "leads.update"
	PermLeadsDelete          Permission = "leads.delete"
	PermCustomersRead        Permission = "customers.read"
	PermCustomersCreate      Permission = "customers.create"
	PermCustomersUpdate      Permission = "customers.update"
	PermCustomersDelete      Permission = "customers.delete"
	PermBookingsRead         Permission = "bookings.read"
	PermBookingsCreate       Permission = "bookings.create"
	PermBookingsUpdate       Permission = "bookings.update"
	PermBookingsDelete       Permission = "bookings.delete"
	PermCleanersRead         Permission = "cleaners.read"
	PermCleanersCreate       Permission = "cleaners.create"
	PermCleanersUpdate       Permission = "cleaners.update"
	PermCleanersDelete       Permission = "cleaners.delete"
	PermServiceRecordsRead   Permission = "service_records.read"
	PermServiceRecordsCreate Permission = "service_records.create"
	PermServiceRecordsUpdate Permission = "service_records.update"
	PermServiceRecordsDelete Permission = "service_records.delete"
	PermPaymentsRead         Permission = "payments.read"
	PermPaymentsCreate       Permission = "payments.create"
	PermPaymentsUpdate       Permission = "payments.update"
	PermInvoicesRead         Permission = "invoices.read"
	PermInvoicesCreate       Permission = "invoices.create"
	PermInvoicesUpdate       Permission = "invoices.update"
	PermReportsRead          Permission = "reports.read"
	PermUsersRead            Permission = "users.read"
	PermUsersManage          Permission = "users.manage"
	PermAuditRead            Permission = "audit.read"
	PermSettingsRead         Permission = "settings.read"
	PermSettingsManage       Permission = "settings.manage"
	PermFeedbackRead         Permission = "feedback.read"
	PermFeedbackCreate       Permission = "feedback.create"
	PermFeedbackUpdate       Permission = "feedback.update"
	PermFeedbackDelete       Permission = "feedback.delete"
	PermNotificationsRead    Permission = "notifications.read"
	PermAttendanceRead       Permission = "attendance.read"
	PermAttendanceManage     Permission = "attendance.manage"
	PermExpensesRead         Permission = "expenses.read"
	PermExpensesManage       Permission = "expenses.manage"
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
		PermCustomersRead, PermCustomersCreate, PermCustomersUpdate, PermCustomersDelete,
		PermBookingsRead, PermBookingsCreate, PermBookingsUpdate, PermBookingsDelete,
		PermCleanersRead, PermCleanersCreate, PermCleanersUpdate, PermCleanersDelete,
		PermServiceRecordsRead, PermServiceRecordsCreate, PermServiceRecordsUpdate, PermServiceRecordsDelete,
		PermPaymentsRead, PermPaymentsCreate, PermPaymentsUpdate,
		PermInvoicesRead, PermInvoicesCreate, PermInvoicesUpdate,
		PermReportsRead,
		PermUsersRead, PermUsersManage,
		PermAuditRead,
		PermSettingsRead, PermSettingsManage,
		PermFeedbackRead, PermFeedbackCreate, PermFeedbackUpdate, PermFeedbackDelete,
		PermNotificationsRead,
		PermAttendanceRead, PermAttendanceManage,
		PermExpensesRead, PermExpensesManage,
	),
	RoleAdmin: perms(
		PermLeadsRead, PermLeadsCreate, PermLeadsUpdate, PermLeadsDelete,
		PermCustomersRead, PermCustomersCreate, PermCustomersUpdate, PermCustomersDelete,
		PermBookingsRead, PermBookingsCreate, PermBookingsUpdate, PermBookingsDelete,
		PermCleanersRead, PermCleanersCreate, PermCleanersUpdate, PermCleanersDelete,
		PermServiceRecordsRead, PermServiceRecordsCreate, PermServiceRecordsUpdate, PermServiceRecordsDelete,
		PermPaymentsRead, PermPaymentsCreate, PermPaymentsUpdate,
		PermInvoicesRead, PermInvoicesCreate, PermInvoicesUpdate,
		PermReportsRead,
		PermUsersRead,
		PermSettingsRead, PermSettingsManage,
		PermFeedbackRead, PermFeedbackCreate, PermFeedbackUpdate, PermFeedbackDelete,
		PermNotificationsRead,
		PermAttendanceRead, PermAttendanceManage,
		PermExpensesRead, PermExpensesManage,
	),
	RoleManager: perms(
		PermLeadsRead, PermLeadsCreate, PermLeadsUpdate,
		PermCustomersRead, PermCustomersCreate, PermCustomersUpdate,
		PermBookingsRead, PermBookingsCreate, PermBookingsUpdate,
		PermCleanersRead, PermCleanersCreate, PermCleanersUpdate,
		PermServiceRecordsRead, PermServiceRecordsCreate, PermServiceRecordsUpdate,
		PermPaymentsRead,
		PermInvoicesRead,
		PermReportsRead,
		PermUsersRead,
		PermSettingsRead,
		PermFeedbackRead,
		PermNotificationsRead,
		PermAttendanceRead, PermAttendanceManage,
		PermExpensesRead,
	),
	RoleDispatch: perms(
		PermLeadsRead, PermLeadsUpdate,
		PermCustomersRead,
		PermBookingsRead, PermBookingsCreate, PermBookingsUpdate,
		PermCleanersRead, PermCleanersUpdate,
		PermServiceRecordsRead, PermServiceRecordsCreate, PermServiceRecordsUpdate,
		PermSettingsRead,
		PermAttendanceRead, PermAttendanceManage,
	),
	RoleAccountant: perms(
		PermLeadsRead,
		PermCustomersRead,
		PermBookingsRead,
		PermCleanersRead,
		PermServiceRecordsRead,
		PermPaymentsRead, PermPaymentsCreate, PermPaymentsUpdate,
		PermInvoicesRead, PermInvoicesCreate, PermInvoicesUpdate,
		PermReportsRead,
		PermSettingsRead,
		PermAttendanceRead,
		PermExpensesRead, PermExpensesManage,
	),
	RoleCleaner: perms(
		PermBookingsRead, PermBookingsUpdate,
		PermServiceRecordsRead, PermServiceRecordsCreate, PermServiceRecordsUpdate,
		PermCustomersRead,
		PermSettingsRead,
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
