package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// CreateStacksResourceHandler creates a handler for getting stacks
func CreateStacksResourceHandler(s *PortainerMCPServer) *ResourceHandler {
	return NewResourceHandler(
		ResourceURIStacks,
		"Portainer Stacks",
		"Lists all available stacks",
		CreateResourceHandler("Stacks", "Lists all available stacks", ResourceURIStacks,
			func(ctx context.Context, request mcp.ReadResourceRequest) (interface{}, error) {
				s.Debug("Handling get stacks request")
				// Stub implementation - real implementation would call s.cli.GetStacks()
				return []map[string]interface{}{
					{"id": 1, "name": "Stack 1", "type": "docker-compose"},
					{"id": 2, "name": "Stack 2", "type": "kubernetes"},
				}, nil
			},
		),
	)
}