package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

func (s *PortainerMCPServer) AddTagFeatures() {
	s.addToolIfExists(ToolListEnvironmentTags, s.HandleGetEnvironmentTags())

	if !s.readOnly {
		s.addToolIfExists(ToolCreateEnvironmentTag, s.HandleCreateEnvironmentTag())
	}
}

func (s *PortainerMCPServer) HandleGetEnvironmentTags() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		environmentTags, err := s.cli.GetEnvironmentTags()
		if err != nil {
			return nil, fmt.Errorf("failed to get environment tags: %w", err)
		}

		data, err := json.Marshal(environmentTags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environment tags: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) HandleCreateEnvironmentTag() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		name, err := parser.GetString("name", true)
		if err != nil {
			return nil, err
		}

		id, err := s.cli.CreateEnvironmentTag(name)
		if err != nil {
			return nil, fmt.Errorf("error creating environment tag. Error: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Environment tag created successfully with ID: %d", id)), nil
	}
}
