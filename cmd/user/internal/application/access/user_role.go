package access

// Role type
type Role uint8

// Roles
const (
	// @TODO: MANAGE YOUR ROLES HERE
	RoleUser Role = 1 << iota
	RoleAdmin
	RoleSuperAdmin
)

func (r Role) String() string {
	return [...]string{"USER", "ADMIN", "SUPER_ADMIN"}[r>>1]
}
