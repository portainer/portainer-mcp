package client

import (
	"fmt"

	"github.com/portainer/portainer-mcp/pkg/portainer/models"
)

// GetEnvironmentTags retrieves all environment tags from the Portainer server.
// Environment tags are the equivalent of Tags in Portainer.
//
// Returns:
//   - A slice of EnvironmentTag objects
//   - An error if the operation fails
func (c *PortainerClient) GetEnvironmentTags() ([]models.EnvironmentTag, error) {
	tags, err := c.cli.ListTags()
	if err != nil {
		return nil, fmt.Errorf("failed to list environment tags: %w", err)
	}

	environmentTags := make([]models.EnvironmentTag, len(tags))
	for i, tag := range tags {
		environmentTags[i] = models.ConvertTagToEnvironmentTag(tag)
	}

	return environmentTags, nil
}

// CreateEnvironmentTag creates a new environment tag on the Portainer server.
// Environment tags are the equivalent of Tags in Portainer.
//
// Parameters:
//   - name: The name of the environment tag
//
// Returns:
//   - The ID of the created environment tag
//   - An error if the operation fails
func (c *PortainerClient) CreateEnvironmentTag(name string) (int, error) {
	id, err := c.cli.CreateTag(name)
	if err != nil {
		return 0, fmt.Errorf("failed to create environment tag: %w", err)
	}

	return int(id), nil
}
