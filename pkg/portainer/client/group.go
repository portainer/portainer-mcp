package client

import (
	"fmt"

	"github.com/deviantony/mcp-go/pkg/portainer/models"
	"github.com/deviantony/mcp-go/pkg/portainer/utils"
)

func (c *PortainerClient) GetEnvironmentGroups() ([]models.Group, error) {
	edgeGroups, err := c.sdkCli.ListEdgeGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to list edge groups: %w", err)
	}

	groups := make([]models.Group, len(edgeGroups))
	for i, eg := range edgeGroups {
		groups[i] = models.ConvertEdgeGroupToGroup(eg)
	}

	return groups, nil
}

func (c *PortainerClient) CreateEnvironmentGroup(name string, environmentIds []int) (int, error) {
	id, err := c.sdkCli.CreateEdgeGroup(name, utils.IntToInt64Slice(environmentIds))
	if err != nil {
		return 0, fmt.Errorf("failed to create environment group: %w", err)
	}

	return int(id), nil
}

func (c *PortainerClient) UpdateEnvironmentGroup(id int, name string, environmentIds []int) error {
	err := c.sdkCli.UpdateEdgeGroup(int64(id), name, utils.IntToInt64Slice(environmentIds))
	if err != nil {
		return fmt.Errorf("failed to update environment group: %w", err)
	}

	return nil
}
