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
//   - name: The name of the access group
//   - environmentIds: The IDs of the environments that are part of the access group
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) CreateAccessGroup(name string, environmentIds []int) (int, error) {
	groupID, err := c.cli.CreateEndpointGroup(name, utils.IntToInt64Slice(environmentIds))
	if err != nil {
		return 0, fmt.Errorf("failed to create access group: %w", err)
	}

	return int(groupID), nil
}

// UpdateAccessGroupName updates the name of an existing access group in Portainer.
//
// Parameters:
//   - id: The ID of the access group
//   - name: The new name for the access group
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateAccessGroupName(id int, name string) error {
	err := c.cli.UpdateEndpointGroup(int64(id), &name, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to update access group name: %w", err)
	}
	return nil
}

// UpdateAccessGroupUserAccesses updates the user access policies of an existing access group in Portainer.
//
// Parameters:
//   - id: The ID of the access group
//   - userAccesses: Map of user IDs to their access level
//
// Valid access levels are:
//   - environment_administrator
//   - helpdesk_user
//   - standard_user
//   - readonly_user
//   - operator_user
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateAccessGroupUserAccesses(id int, userAccesses map[int]string) error {
	uac := utils.IntToInt64Map(userAccesses)
	err := c.cli.UpdateEndpointGroup(int64(id), nil, &uac, nil)
	if err != nil {
		return fmt.Errorf("failed to update access group user accesses: %w", err)
	}
	return nil
}

// UpdateAccessGroupTeamAccesses updates the team access policies of an existing access group in Portainer.
//
// Parameters:
//   - id: The ID of the access group
//   - teamAccesses: Map of team IDs to their access level
//
// Valid access levels are:
//   - environment_administrator
//   - helpdesk_user
//   - standard_user
//   - readonly_user
//   - operator_user
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateAccessGroupTeamAccesses(id int, teamAccesses map[int]string) error {
	tac := utils.IntToInt64Map(teamAccesses)
	err := c.cli.UpdateEndpointGroup(int64(id), nil, nil, &tac)
	if err != nil {
		return fmt.Errorf("failed to update access group team accesses: %w", err)
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
