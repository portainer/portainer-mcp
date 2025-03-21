package client

import (
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
)

// GetUsers retrieves all users from the Portainer server.
//
// Returns:
//   - A slice of User objects containing user information
//   - An error if the operation fails
func (c *PortainerClient) GetUsers() ([]models.User, error) {
	portainerUsers, err := c.cli.ListUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]models.User, len(portainerUsers))
	for i, user := range portainerUsers {
		users[i] = models.ConvertToUser(user)
	}

	return users, nil
}

// UpdateUser updates the role of a user.
//
// Parameters:
//   - id: The ID of the user to update
//   - role: The new role for the user. Must be one of: admin, user, edge_admin
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateUser(id int, role string) error {
	roleInt := convertRole(role)
	if roleInt == 0 {
		return fmt.Errorf("invalid role: must be admin, user or edge_admin")
	}

	return c.cli.UpdateUser(id, roleInt)
}

func convertRole(role string) int64 {
	switch role {
	case models.UserRoleAdmin:
		return 1
	case models.UserRoleUser:
		return 2
	case models.UserRoleEdgeAdmin:
		return 3
	default:
		return 0
	}
}
