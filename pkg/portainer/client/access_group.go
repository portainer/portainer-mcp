package client

import (
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/pkg/portainer/utils"
)

// GetAccessGroups retrieves all access groups from the Portainer server.
// Access groups are the equivalent of Endpoint Groups in Portainer.
//
// Returns:
//   - A slice of AccessGroup objects
//   - An error if the operation fails
func (c *PortainerClient) GetAccessGroups() ([]models.AccessGroup, error) {
	groups, err := c.cli.ListEndpointGroups()
	if err != nil {
		return nil, err
	}

	endpoints, err := c.cli.ListEndpoints()
	if err != nil {
		return nil, err
	}

	accessGroups := make([]models.AccessGroup, len(groups))
	for i, group := range groups {
		accessGroups[i] = models.ConvertEndpointGroupToAccessGroup(group, endpoints)
	}

	return accessGroups, nil
}

// CreateAccessGroup creates a new access group in Portainer.
//
// Parameters:
//   - accessGroup: The AccessGroup object containing the name, environment IDs, team access policies, and user access policies.
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) CreateAccessGroup(accessGroup models.AccessGroup) (int, error) {
	groupID, err := c.cli.CreateEndpointGroup(accessGroup.Name, utils.IntToInt64Slice(accessGroup.EnvironmentIds))
	if err != nil {
		return 0, fmt.Errorf("failed to create access group: %w", err)
	}

	err = c.cli.UpdateEndpointGroup(groupID,
		accessGroup.Name,
		utils.IntToInt64Map(accessGroup.TeamAccesses),
		utils.IntToInt64Map(accessGroup.UserAccesses),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to update access group: %w", err)
	}

	return int(groupID), nil
}

// UpdateAccessGroup updates an existing access group in Portainer.
//
// Parameters:
//   - accessGroup: The AccessGroup object containing the name, environment IDs, team access policies, and user access policies.
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateAccessGroup(accessGroup models.AccessGroup) error {
	err := c.cli.UpdateEndpointGroup(int64(accessGroup.ID),
		accessGroup.Name,
		utils.IntToInt64Map(accessGroup.TeamAccesses),
		utils.IntToInt64Map(accessGroup.UserAccesses),
	)
	if err != nil {
		return fmt.Errorf("failed to update access group: %w", err)
	}
	return nil
}

// AddEnvironmentToAccessGroup adds an environment to an access group
//
// Parameters:
//   - id: The ID of the access group
//   - environmentId: The ID of the environment to add to the access group
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) AddEnvironmentToAccessGroup(id int, environmentId int) error {
	return c.cli.AddEnvironmentToEndpointGroup(int64(id), int64(environmentId))
}

// RemoveEnvironmentFromAccessGroup removes an environment from an access group
//
// Parameters:
//   - id: The ID of the access group
//   - environmentId: The ID of the environment to remove from the access group
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) RemoveEnvironmentFromAccessGroup(id int, environmentId int) error {
	return c.cli.RemoveEnvironmentFromEndpointGroup(int64(id), int64(environmentId))
}
