package auth

// Admin role constants.
const (
	RoleViewer     = "viewer"
	RoleAdmin      = "admin"
	RoleSuperAdmin = "superadmin"
)

// AllAdminRoles returns all valid admin roles.
func AllAdminRoles() []string {
	return []string{RoleViewer, RoleAdmin, RoleSuperAdmin}
}

// WriteRoles returns roles that can modify data.
func WriteRoles() []string {
	return []string{RoleAdmin, RoleSuperAdmin}
}
