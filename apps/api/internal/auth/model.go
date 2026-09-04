package auth

type Role string

const (
	RoleSuperAdmin Role = "SUPER_ADMIN"
	RoleAdmin      Role = "ADMIN"
	RoleManager    Role = "MANAGER"
	RoleSales      Role = "SALES"
	RoleStaff      Role = "STAFF"
	RoleAccountant Role = "ACCOUNTANT"
)

func (r Role) Valid() bool {
	switch r {
	case RoleSuperAdmin, RoleAdmin, RoleManager, RoleSales, RoleStaff, RoleAccountant:
		return true
	}
	return false
}

type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	Role         Role
}
