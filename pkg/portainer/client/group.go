package client

import (
	"fmt"

	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/portainer/portainer-mcp/pkg/portainer/utils"
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

// UpdateEnvironmentGroupName updates the name of an existing environment group.
// Environment groups are the equivalent of Edge Groups in Portainer.
//
// Parameters:
//   - id: The ID of the environment group to update
//   - name: The new name for the environment group
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateEnvironmentGroupName(id int, name string) error {
	err := c.cli.UpdateEdgeGroup(int64(id), &name, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to update environment group name: %w", err)
	}
	return nil
}

// UpdateEnvironmentGroupEnvironments updates the environments associated with an environment group.
// Environment groups are the equivalent of Edge Groups in Portainer.
//
// Parameters:
//   - id: The ID of the environment group to update
//   - environmentIds: A slice of environment IDs to include in the group
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateEnvironmentGroupEnvironments(id int, environmentIds []int) error {
	envs := utils.IntToInt64Slice(environmentIds)
	err := c.cli.UpdateEdgeGroup(int64(id), nil, &envs, nil)
	if err != nil {
		return fmt.Errorf("failed to update environment group environments: %w", err)
	}
	return nil
}

// UpdateEnvironmentGroupTags updates the tags associated with an environment group.
// Environment groups are the equivalent of Edge Groups in Portainer.
//
// Parameters:
//   - id: The ID of the environment group to update
//   - tagIds: A slice of tag IDs to include in the group
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateEnvironmentGroupTags(id int, tagIds []int) error {
	tags := utils.IntToInt64Slice(tagIds)
	err := c.cli.UpdateEdgeGroup(int64(id), nil, nil, &tags)
	if err != nil {
		return fmt.Errorf("failed to update environment group tags: %w", err)
	}
	return nil
}
