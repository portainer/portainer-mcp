package client

import (
	"fmt"

	"github.com/deviantony/mcp-go/pkg/portainer/models"
)

func (c *PortainerClient) GetEnvironments() ([]models.Environment, error) {
	endpoints, err := c.sdkCli.ListEndpoints()
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}

	environments := make([]models.Environment, len(endpoints))
	for i, endpoint := range endpoints {
		environments[i] = models.ConvertEndpointToEnvironment(endpoint)
	}

	return environments, nil
}

func (c *PortainerClient) UpdateEnvironment(id int, tagIds []int) error {
	tagIdsInt64 := make([]int64, len(tagIds))
	for i, tagId := range tagIds {
		tagIdsInt64[i] = int64(tagId)
	}

	err := c.sdkCli.UpdateEndpoint(int64(id), tagIdsInt64)
	if err != nil {
		return fmt.Errorf("failed to update environment: %w", err)
	}

	return nil
}
