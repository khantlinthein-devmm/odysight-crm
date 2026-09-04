package auth

type Role string

const (
	RoleSuperAdmin Role = "SUPER_ADMIN"
	RoleAdmin      Role = "ADMIN"
	RoleManager    Role = "MANAGER"
	RoleDispatch   Role = "DISPATCH"
	RoleAccountant Role = "ACCOUNTANT"
	RoleCleaner    Role = "CLEANER"
)

func (r Role) Valid() bool {
	switch r {
	case RoleSuperAdmin, RoleAdmin, RoleManager, RoleDispatch, RoleAccountant, RoleCleaner:
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
