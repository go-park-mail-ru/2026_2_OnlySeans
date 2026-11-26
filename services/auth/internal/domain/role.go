package domain

type Role string

const (
	RoleReader Role = "reader"
	RoleEditor Role = "editor"
	RoleAdmin  Role = "admin"
)

func (role *Role) CanEditPersons() bool {
	switch *role {
	case RoleAdmin, RoleEditor:
		return true
	default:
		return false
	}
}
