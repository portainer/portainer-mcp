package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/toolgen"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddEnvironmentGroupFeatures() {
	s.addToolIfExists(ToolCreateEnvironmentGroup, s.handleCreateEnvironmentGroup())
	s.addToolIfExists(ToolListEnvironmentGroups, s.handleGetEnvironmentGroups())
	s.addToolIfExists(ToolUpdateEnvironmentGroup, s.handleUpdateEnvironmentGroup())
}

func (s *PortainerMCPServer) handleGetEnvironmentGroups() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		edgeGroups, err := s.cli.GetEnvironmentGroups()
		if err != nil {
			return nil, fmt.Errorf("failed to get environment groups: %w", err)
		}

		data, err := json.Marshal(edgeGroups)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environment groups: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) handleCreateEnvironmentGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		name, err := parser.GetString("name", true)
		if err != nil {
			return nil, err
		}

		environmentIds, err := parser.GetArrayOfIntegers("environmentIds", true)
		if err != nil {
			return nil, err
		}

		id, err := s.cli.CreateEnvironmentGroup(name, environmentIds)
		if err != nil {
			return nil, fmt.Errorf("error creating environment group. Error: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Environment group created successfully with ID: %d", id)), nil
	}
}

func (s *PortainerMCPServer) handleUpdateEnvironmentGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		name, err := parser.GetString("name", true)
		if err != nil {
			return nil, err
		}

		environmentIds, err := parser.GetArrayOfIntegers("environmentIds", false)
		if err != nil {
			return nil, err
		}

		tagIds, err := parser.GetArrayOfIntegers("tagIds", false)
		if err != nil {
			return nil, err
		}

		err = s.cli.UpdateEnvironmentGroup(id, name, environmentIds, tagIds)
		if err != nil {
			return nil, fmt.Errorf("error updating environment group. Error: %w", err)
		}

		return mcp.NewToolResultText("Environment group updated successfully"), nil
	}
}
