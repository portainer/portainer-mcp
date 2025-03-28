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
	s.addToolIfExists(ToolUpdateEnvironmentGroupName, s.handleUpdateEnvironmentGroupName())
	s.addToolIfExists(ToolUpdateEnvironmentGroupEnvironments, s.handleUpdateEnvironmentGroupEnvironments())
	s.addToolIfExists(ToolUpdateEnvironmentGroupTags, s.handleUpdateEnvironmentGroupTags())
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

func (s *PortainerMCPServer) handleUpdateEnvironmentGroupName() server.ToolHandlerFunc {
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

		err = s.cli.UpdateEnvironmentGroupName(id, name)
		if err != nil {
			return nil, fmt.Errorf("failed to update environment group name: %w", err)
		}

		return mcp.NewToolResultText("Environment group name updated successfully"), nil
	}
}

func (s *PortainerMCPServer) handleUpdateEnvironmentGroupEnvironments() server.ToolHandlerFunc {
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

		environmentIds, err := parser.GetArrayOfIntegers("environmentIds", true)
		if err != nil {
			return nil, err
		}

		err = s.cli.UpdateEnvironmentGroupEnvironments(id, name, environmentIds)
		if err != nil {
			return nil, fmt.Errorf("failed to update environment group environments: %w", err)
		}

		return mcp.NewToolResultText("Environment group environments updated successfully"), nil
	}
}

func (s *PortainerMCPServer) handleUpdateEnvironmentGroupTags() server.ToolHandlerFunc {
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

		tagIds, err := parser.GetArrayOfIntegers("tagIds", true)
		if err != nil {
			return nil, err
		}

		err = s.cli.UpdateEnvironmentGroupTags(id, name, tagIds)
		if err != nil {
			return nil, fmt.Errorf("failed to update environment group tags: %w", err)
		}

		return mcp.NewToolResultText("Environment group tags updated successfully"), nil
	}
}
