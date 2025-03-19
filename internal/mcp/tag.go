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

	createTagTool := mcp.NewTool("createEnvironmentTag",
		mcp.WithDescription("Create a new environment tag"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the tag"),
		),
	)

	s.srv.AddResource(environmentTagsResource, s.handleGetEnvironmentTags())
	s.srv.AddTool(createTagTool, s.handleCreateEnvironmentTag())
}

func (s *PortainerMCPServer) handleGetEnvironmentTags() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]interface{}, error) {
		environmentTags, err := s.cli.GetEnvironmentTags()
		if err != nil {
			return nil, fmt.Errorf("failed to get environment tags: %w", err)
		}

		data, err := json.Marshal(environmentTags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environment tags: %w", err)
		}

		return []interface{}{
			mcp.TextResourceContents{
				ResourceContents: mcp.ResourceContents{
					URI:      "portainer://environment-tags",
					MIMEType: "application/json",
				},
				Text: string(data),
			},
		}, nil
	}
}

func (s *PortainerMCPServer) handleCreateEnvironmentTag() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, ok := request.Params.Arguments["name"].(string)
		if !ok {
			return mcp.NewToolResultError("tag name is required"), nil
		}

		id, err := s.cli.CreateEnvironmentTag(name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error creating environment tag: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Environment tag created successfully with ID: %d", id)), nil
	}
}
