package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// CreateTagsResourceHandler creates a handler for getting tags
func CreateTagsResourceHandler(s *PortainerMCPServer) *ResourceHandler {
	return NewResourceHandler(
		ResourceURITags,
		"Portainer Tags",
		"Lists all available tags",
		CreateResourceHandler("Tags", "Lists all available tags", ResourceURITags,
			func(ctx context.Context, request mcp.ReadResourceRequest) (interface{}, error) {
				s.Debug("Handling get tags request")
				// Stub implementation - real implementation would call s.cli.GetTags()
				return []map[string]interface{}{
					{"id": 1, "name": "Tag 1"},
					{"id": 2, "name": "Tag 2"},
				}, nil
			},
		),
	)
}