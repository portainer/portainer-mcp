package models

import (
	"github.com/portainer/client-api-go/v2/pkg/models"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// User role constants
const (
	UserRoleAdmin     = "admin"
	UserRoleUser      = "user"
	UserRoleEdgeAdmin = "edge_admin"
	UserRoleUnknown   = "unknown"
)

func ConvertToUser(u *models.PortainereeUser) User {
	return User{
		ID:       int(u.ID),
		Username: u.Username,
		Role:     convertUserRole(u),
	}
}

func convertUserRole(u *models.PortainereeUser) string {
	switch u.Role {
	case 1:
		return UserRoleAdmin
	case 2:
		return UserRoleUser
	case 3:
		return UserRoleEdgeAdmin
	default:
		return UserRoleUnknown
	}
}
