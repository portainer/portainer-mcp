package client

import (
	"fmt"
	"strings"

	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/portainer/portainer-mcp/pkg/portainer/utils"
)

// GetStacks retrieves all stacks from the Portainer server.
// Stacks are the equivalent of Edge Stacks in Portainer.
//
// Returns:
//   - A slice of Stack objects
//   - An error if the operation fails
func (c *PortainerClient) GetStacks() ([]models.Stack, error) {
	edgeStacks, err := c.cli.ListEdgeStacks()
	if err != nil {
		// Edge Compute features may be disabled, returning a 503.
		// Return an empty list instead of failing the entire request.
		if isEdgeComputeDisabledError(err) {
			return []models.Stack{}, nil
		}
		return nil, fmt.Errorf("failed to list edge stacks: %w", err)
	}

	stacks := make([]models.Stack, len(edgeStacks))
	for i, es := range edgeStacks {
		stacks[i] = models.ConvertEdgeStackToStack(es)
	}

	return stacks, nil
}

// isEdgeComputeDisabledError checks if an error is caused by Edge Compute
// features being disabled in Portainer (typically a 503 Service Unavailable).
func isEdgeComputeDisabledError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "503") || strings.Contains(errStr, "Edge Compute features are disabled")
}

// GetStackFile retrieves the file content of a stack from the Portainer server.
// Stacks are the equivalent of Edge Stacks in Portainer.
//
// Parameters:
//   - id: The ID of the stack to retrieve
//
// Returns:
//   - The file content of the stack (Compose file)
//   - An error if the operation fails
func (c *PortainerClient) GetStackFile(id int) (string, error) {
	file, err := c.cli.GetEdgeStackFile(int64(id))
	if err != nil {
		if isEdgeComputeDisabledError(err) {
			return "", fmt.Errorf("edge stacks are not available (Edge Compute is disabled); use getLocalStackFile for local stacks")
		}
		return "", fmt.Errorf("failed to get edge stack file: %w", err)
	}

	return file, nil
}

// CreateStack creates a new stack on the Portainer server.
// This function specifically creates a Docker Compose stack.
// Stacks are the equivalent of Edge Stacks in Portainer.
//
// Parameters:
//   - name: The name of the stack
//   - file: The file content of the stack (Compose file)
//   - environmentGroupIds: A slice of environment group IDs to include in the stack
//
// Returns:
//   - The ID of the created stack
//   - An error if the operation fails
func (c *PortainerClient) CreateStack(name, file string, environmentGroupIds []int) (int, error) {
	id, err := c.cli.CreateEdgeStack(name, file, utils.IntToInt64Slice(environmentGroupIds))
	if err != nil {
		return 0, fmt.Errorf("failed to create edge stack: %w", err)
	}

	return int(id), nil
}

// UpdateStack updates an existing stack on the Portainer server.
// This function specifically updates a Docker Compose stack.
// Stacks are the equivalent of Edge Stacks in Portainer.
//
// Parameters:
//   - id: The ID of the stack to update
//   - file: The file content of the stack (Compose file)
//   - environmentGroupIds: A slice of environment group IDs to include in the stack
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateStack(id int, file string, environmentGroupIds []int) error {
	err := c.cli.UpdateEdgeStack(int64(id), file, utils.IntToInt64Slice(environmentGroupIds))
	if err != nil {
		return fmt.Errorf("failed to update edge stack: %w", err)
	}

	return nil
}
