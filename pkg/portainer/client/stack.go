package client

import (
	"fmt"

	"github.com/deviantony/mcp-go/pkg/portainer/models"
	"github.com/deviantony/mcp-go/pkg/portainer/utils"
)

func (c *PortainerClient) GetStacks() ([]models.Stack, error) {
	edgeStacks, err := c.sdkCli.ListEdgeStacks()
	if err != nil {
		return nil, fmt.Errorf("failed to list edge stacks: %w", err)
	}

	stacks := make([]models.Stack, len(edgeStacks))
	for i, es := range edgeStacks {
		stacks[i] = models.ConvertEdgeStackToStack(es)
	}

	return stacks, nil
}

func (c *PortainerClient) GetStackFile(id int) (string, error) {
	file, err := c.sdkCli.GetEdgeStackFile(int64(id))
	if err != nil {
		return "", fmt.Errorf("failed to get edge stack file: %w", err)
	}

	return file, nil
}

func (c *PortainerClient) CreateStack(name, file string, environmentGroupIds []int) (int, error) {
	id, err := c.sdkCli.CreateEdgeStack(name, file, utils.IntToInt64Slice(environmentGroupIds))
	if err != nil {
		return 0, fmt.Errorf("failed to create edge stack: %w", err)
	}

	return int(id), nil
}

func (c *PortainerClient) UpdateStack(id int, file string, environmentGroupIds []int) error {
	err := c.sdkCli.UpdateEdgeStack(int64(id), file, utils.IntToInt64Slice(environmentGroupIds))
	if err != nil {
		return fmt.Errorf("failed to update edge stack: %w", err)
	}

	return nil
}
