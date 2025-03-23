package client

import "github.com/deviantony/portainer-mcp/pkg/portainer/models"

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
