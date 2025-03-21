package client

import (
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/pkg/portainer/utils"
)

// GetEnvironments retrieves all environments from the Portainer server.
//
// Returns:
//   - A slice of Environment objects
//   - An error if the operation fails
func (c *PortainerClient) GetEnvironments() ([]models.Environment, error) {
	endpoints, err := c.cli.ListEndpoints()
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}

	environments := make([]models.Environment, len(endpoints))
	for i, endpoint := range endpoints {
		environments[i] = models.ConvertEndpointToEnvironment(endpoint)
	}

	return environments, nil
}

// UpdateEnvironment updates the tags for an environment.
//
// Parameters:
//   - id: The ID of the environment to update
//   - tagIds: A slice of tag IDs to associate with the environment
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateEnvironment(id int, tagIds []int) error {
	err := c.cli.UpdateEndpoint(int64(id), utils.IntToInt64Slice(tagIds))
	if err != nil {
		return fmt.Errorf("failed to update environment: %w", err)
	}

	return nil
}
