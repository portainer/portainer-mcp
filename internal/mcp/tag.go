package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddTagFeatures() {
	environmentTagsResource := mcp.NewResource("portainer://environment-tags",
		"Portainer Environment Tags",
		mcp.WithResourceDescription("Lists all available environment tags"),
		mcp.WithMIMEType("application/json"),
	)

	s.srv.AddResource(environmentTagsResource, s.handleGetEnvironmentTags())

	createEnvironmentTagTool := s.tools[ToolCreateEnvironmentTag]
	s.srv.AddTool(createEnvironmentTagTool, s.handleCreateEnvironmentTag())
}

func (s *PortainerMCPServer) handleGetEnvironmentTags() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		environmentTags, err := s.cli.GetEnvironmentTags()
		if err != nil {
			return nil, fmt.Errorf("failed to get environment tags: %w", err)
		}

		data, err := json.Marshal(environmentTags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environment tags: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "portainer://environment-tags",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	}
}

func (s *PortainerMCPServer) handleCreateEnvironmentTag() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, ok := request.Params.Arguments["name"].(string)
		if !ok {
			return nil, fmt.Errorf("tag name is required")
		}

		id, err := s.cli.CreateEnvironmentTag(name)
		if err != nil {
			return nil, fmt.Errorf("error creating environment tag. Error: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Environment tag created successfully with ID: %d", id)), nil
	}
}
