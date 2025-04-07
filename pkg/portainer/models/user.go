package models

import (
	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
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

func ConvertToUser(rawUser *apimodels.PortainereeUser) User {
	return User{
		ID:       int(rawUser.ID),
		Username: rawUser.Username,
		Role:     convertUserRole(rawUser),
	}
}

func convertUserRole(rawUser *apimodels.PortainereeUser) string {
	switch rawUser.Role {
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
