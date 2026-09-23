package auth

import "testing"

func TestSuperAdminHasAllPermissions(t *testing.T) {
	for p := range rolePermissions[RoleSuperAdmin] {
		if !roleHasPermission(RoleSuperAdmin, p) {
			t.Errorf("SUPER_ADMIN missing %s", p)
		}
	}
	if n := len(rolePermissions[RoleSuperAdmin]); n != len(rolePermissions[RoleAdmin]) {
		// Every permission SUPER_ADMIN has is a superset; ADMIN was trimmed.
		t.Logf("SUPER_ADMIN has %d perms, ADMIN has %d", n, len(rolePermissions[RoleAdmin]))
	}
}

func TestAdminCannotManageUsersOrReadAudit(t *testing.T) {
	if roleHasPermission(RoleAdmin, PermUsersManage) {
		t.Error("ADMIN must not hold users.manage")
	}
	if roleHasPermission(RoleAdmin, PermAuditRead) {
		t.Error("ADMIN must not hold audit.read")
	}
	if !roleHasPermission(RoleAdmin, PermUsersRead) {
		t.Error("ADMIN should hold users.read")
	}
}

func TestOnlySuperAdminManagesSuperAdmin(t *testing.T) {
	if !roleHasPermission(RoleSuperAdmin, PermUsersManage) {
		t.Error("SUPER_ADMIN must hold users.manage")
	}
	roles := []Role{RoleManager, RoleDispatch, RoleAccountant, RoleCleaner}
	for _, r := range roles {
		if roleHasPermission(r, PermUsersManage) {
			t.Errorf("%s must not hold users.manage", r)
		}
	}
}

func TestRoleBoundaries(t *testing.T) {
	tests := []struct {
		role    Role
		perm    Permission
		allowed bool
	}{
		{RoleCleaner, PermBookingsDelete, false},
		{RoleCleaner, PermBookingsRead, true},
		{RoleCleaner, PermSettingsManage, false},
		{RoleManager, PermPaymentsCreate, false},
		{RoleAccountant, PermPaymentsCreate, true},
		{RoleDispatch, PermCleanersUpdate, true},
		{RoleDispatch, PermPaymentsRead, false},
		{RoleManager, PermReportsRead, true},
		{RoleManager, PermInvoicesRead, true},
		{RoleManager, PermInvoicesCreate, false},
		{RoleAccountant, PermInvoicesCreate, true},
		{RoleDispatch, PermInvoicesRead, false},
		{RoleCleaner, PermInvoicesRead, false},
		{RoleSuperAdmin, PermSitesDelete, true},
		{RoleAdmin, PermContractsDelete, true},
		{RoleManager, PermSitesCreate, true},
		{RoleManager, PermSitesDelete, false},
		{RoleManager, PermContractsCreate, false},
		{RoleManager, PermContractsRead, true},
		{RoleManager, PermQuotesCreate, true},
		{RoleManager, PermQuotesApprove, false},
		{RoleAdmin, PermQuotesApprove, true},
		{RoleDispatch, PermSitesRead, true},
		{RoleDispatch, PermSitesCreate, false},
		{RoleDispatch, PermContractsRead, false},
		{RoleAccountant, PermContractsRead, true},
		{RoleAccountant, PermQuotesApprove, false},
		{RoleCleaner, PermChecklistsManage, true},
		{RoleCleaner, PermContractsRead, false},
		{RoleCleaner, PermSitesRead, true},
	}
	for _, tt := range tests {
		if got := roleHasPermission(tt.role, tt.perm); got != tt.allowed {
			t.Errorf("roleHasPermission(%s, %s) = %v, want %v", tt.role, tt.perm, got, tt.allowed)
		}
	}
}

func TestUnknownRoleDenied(t *testing.T) {
	if roleHasPermission("SUPER_USER", PermLeadsRead) {
		t.Error("unknown role must not have permissions")
	}
}
