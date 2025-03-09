package client

import (
	"fmt"

	"github.com/deviantony/mcp-go/pkg/portainer/models"
)

func (c *PortainerClient) GetEnvironmentTags() ([]models.EnvironmentTag, error) {
	tags, err := c.sdkCli.ListTags()
	if err != nil {
		return nil, fmt.Errorf("failed to list environment tags: %w", err)
	}

	environmentTags := make([]models.EnvironmentTag, len(tags))
	for i, tag := range tags {
		environmentTags[i] = models.ConvertTagToEnvironmentTag(tag)
	}

	return environmentTags, nil
}

func (c *PortainerClient) CreateEnvironmentTag(name string) (int, error) {
	id, err := c.sdkCli.CreateTag(name)
	if err != nil {
		return 0, fmt.Errorf("failed to create environment tag: %w", err)
	}

	return int(id), nil
}
