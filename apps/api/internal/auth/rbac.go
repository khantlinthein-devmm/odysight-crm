package auth

// Permission constants follow the "resource.action" convention.
const (
	PermLeadsRead          Permission = "leads.read"
	PermLeadsCreate        Permission = "leads.create"
	PermLeadsUpdate        Permission = "leads.update"
	PermLeadsDelete        Permission = "leads.delete"
	PermCustomersRead      Permission = "customers.read"
	PermCustomersCreate    Permission = "customers.create"
	PermCustomersUpdate    Permission = "customers.update"
	PermCustomersDelete    Permission = "customers.delete"
	PermBookingsRead       Permission = "bookings.read"
	PermBookingsCreate     Permission = "bookings.create"
	PermBookingsUpdate     Permission = "bookings.update"
	PermBookingsDelete     Permission = "bookings.delete"
	PermCleanersRead       Permission = "cleaners.read"
	PermCleanersCreate     Permission = "cleaners.create"
	PermCleanersUpdate     Permission = "cleaners.update"
	PermCleanersDelete     Permission = "cleaners.delete"
	PermServiceRecordsRead   Permission = "service_records.read"
	PermServiceRecordsCreate Permission = "service_records.create"
	PermServiceRecordsUpdate Permission = "service_records.update"
	PermServiceRecordsDelete Permission = "service_records.delete"
	PermPaymentsRead       Permission = "payments.read"
	PermPaymentsCreate     Permission = "payments.create"
	PermPaymentsUpdate     Permission = "payments.update"
	PermReportsRead        Permission = "reports.read"
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
		PermReportsRead,
	),
	RoleAdmin: perms(
		PermLeadsRead, PermLeadsCreate, PermLeadsUpdate, PermLeadsDelete,
		PermCustomersRead, PermCustomersCreate, PermCustomersUpdate, PermCustomersDelete,
		PermBookingsRead, PermBookingsCreate, PermBookingsUpdate, PermBookingsDelete,
		PermCleanersRead, PermCleanersCreate, PermCleanersUpdate, PermCleanersDelete,
		PermServiceRecordsRead, PermServiceRecordsCreate, PermServiceRecordsUpdate, PermServiceRecordsDelete,
		PermPaymentsRead, PermPaymentsCreate, PermPaymentsUpdate,
		PermReportsRead,
	),
	RoleManager: perms(
		PermLeadsRead, PermLeadsCreate, PermLeadsUpdate,
		PermCustomersRead, PermCustomersCreate, PermCustomersUpdate,
		PermBookingsRead, PermBookingsCreate, PermBookingsUpdate,
		PermCleanersRead, PermCleanersCreate, PermCleanersUpdate,
		PermServiceRecordsRead, PermServiceRecordsCreate, PermServiceRecordsUpdate,
		PermPaymentsRead,
		PermReportsRead,
	),
	RoleDispatch: perms(
		PermLeadsRead, PermLeadsUpdate,
		PermCustomersRead,
		PermBookingsRead, PermBookingsCreate, PermBookingsUpdate,
		PermCleanersRead, PermCleanersUpdate,
		PermServiceRecordsRead, PermServiceRecordsCreate, PermServiceRecordsUpdate,
	),
	RoleAccountant: perms(
		PermLeadsRead,
		PermCustomersRead,
		PermBookingsRead,
		PermCleanersRead,
		PermServiceRecordsRead,
		PermPaymentsRead, PermPaymentsCreate, PermPaymentsUpdate,
		PermReportsRead,
	),
	RoleCleaner: perms(
		PermBookingsRead, PermBookingsUpdate,
		PermServiceRecordsRead, PermServiceRecordsCreate, PermServiceRecordsUpdate,
		PermCustomersRead,
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
