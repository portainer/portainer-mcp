package client

import (
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/pkg/portainer/utils"
)

// GetEnvironmentGroups retrieves all environment groups from the Portainer server.
// Environment groups are the equivalent of Edge Groups in Portainer.
//
// Returns:
//   - A slice of Group objects
//   - An error if the operation fails
func (c *PortainerClient) GetEnvironmentGroups() ([]models.Group, error) {
	edgeGroups, err := c.cli.ListEdgeGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to list edge groups: %w", err)
	}

	groups := make([]models.Group, len(edgeGroups))
	for i, eg := range edgeGroups {
		groups[i] = models.ConvertEdgeGroupToGroup(eg)
	}

	return groups, nil
}

// CreateEnvironmentGroup creates a new environment group on the Portainer server.
// Environment groups are the equivalent of Edge Groups in Portainer.
// Parameters:
//   - name: The name of the environment group
//   - environmentIds: A slice of environment IDs to include in the group
//
// Returns:
//   - The ID of the created environment group
//   - An error if the operation fails
func (c *PortainerClient) CreateEnvironmentGroup(name string, environmentIds []int) (int, error) {
	id, err := c.cli.CreateEdgeGroup(name, utils.IntToInt64Slice(environmentIds))
	if err != nil {
		return 0, fmt.Errorf("failed to create environment group: %w", err)
	}

	return int(id), nil
}

// UpdateEnvironmentGroup updates an existing environment group on the Portainer server.
// Environment groups are the equivalent of Edge Groups in Portainer.
//
// Parameters:
//   - id: The ID of the environment group to update
//   - name: The new name for the environment group
//   - environmentIds: A slice of environment IDs to include in the group
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateEnvironmentGroup(id int, name string, environmentIds []int) error {
	err := c.cli.UpdateEdgeGroup(int64(id), name, utils.IntToInt64Slice(environmentIds))
	if err != nil {
		return fmt.Errorf("failed to update environment group: %w", err)
	}

	return nil
}
