package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

func (s *PortainerMCPServer) AddEnvironmentGroupFeatures() {
	s.addToolIfExists(ToolListEnvironmentGroups, s.HandleGetEnvironmentGroups())

	if !s.readOnly {
		s.addToolIfExists(ToolCreateEnvironmentGroup, s.HandleCreateEnvironmentGroup())
		s.addToolIfExists(ToolUpdateEnvironmentGroupName, s.HandleUpdateEnvironmentGroupName())
		s.addToolIfExists(ToolUpdateEnvironmentGroupEnvironments, s.HandleUpdateEnvironmentGroupEnvironments())
		s.addToolIfExists(ToolUpdateEnvironmentGroupTags, s.HandleUpdateEnvironmentGroupTags())
	}
}

func (s *PortainerMCPServer) HandleGetEnvironmentGroups() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleCreateEnvironmentGroup() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleUpdateEnvironmentGroupName() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleUpdateEnvironmentGroupEnvironments() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleUpdateEnvironmentGroupTags() server.ToolHandlerFunc {
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
