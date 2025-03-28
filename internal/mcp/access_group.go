package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/toolgen"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddAccessGroupFeatures() {
	s.addToolIfExists(ToolCreateAccessGroup, s.handleCreateAccessGroup())
	s.addToolIfExists(ToolListAccessGroups, s.handleGetAccessGroups())
	s.addToolIfExists(ToolUpdateAccessGroupName, s.handleUpdateAccessGroupName())
	s.addToolIfExists(ToolUpdateAccessGroupUserAccesses, s.handleUpdateAccessGroupUserAccesses())
	s.addToolIfExists(ToolUpdateAccessGroupTeamAccesses, s.handleUpdateAccessGroupTeamAccesses())
	s.addToolIfExists(ToolAddEnvironmentToAccessGroup, s.handleAddEnvironmentToAccessGroup())
	s.addToolIfExists(ToolRemoveEnvironmentFromAccessGroup, s.handleRemoveEnvironmentFromAccessGroup())
}

func (s *PortainerMCPServer) handleGetAccessGroups() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		accessGroups, err := s.cli.GetAccessGroups()
		if err != nil {
			return nil, fmt.Errorf("failed to get access groups: %w", err)
		}

		data, err := json.Marshal(accessGroups)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal access groups: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) handleCreateAccessGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		name, err := parser.GetString("name", true)
		if err != nil {
			return nil, err
		}

		environmentIds, err := parser.GetArrayOfIntegers("environmentIds", false)
		if err != nil {
			return nil, err
		}

		groupID, err := s.cli.CreateAccessGroup(name, environmentIds)
		if err != nil {
			return nil, fmt.Errorf("failed to create access group: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Access group created successfully with ID: %d", groupID)), nil
	}
}

func (s *PortainerMCPServer) handleUpdateAccessGroupName() server.ToolHandlerFunc {
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

		err = s.cli.UpdateAccessGroupName(id, name)
		if err != nil {
			return nil, fmt.Errorf("failed to update access group name: %w", err)
		}

		return mcp.NewToolResultText("Access group name updated successfully"), nil
	}
}

func (s *PortainerMCPServer) handleUpdateAccessGroupUserAccesses() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		userAccesses, err := parser.GetArrayOfObjects("userAccesses", true)
		if err != nil {
			return nil, err
		}

		userAccessesMap, err := parseAccessMap(userAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid user accesses: %w", err)
		}

		err = s.cli.UpdateAccessGroupUserAccesses(id, userAccessesMap)
		if err != nil {
			return nil, fmt.Errorf("failed to update access group user accesses: %w", err)
		}

		return mcp.NewToolResultText("Access group user accesses updated successfully"), nil
	}
}

func (s *PortainerMCPServer) handleUpdateAccessGroupTeamAccesses() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		teamAccesses, err := parser.GetArrayOfObjects("teamAccesses", true)
		if err != nil {
			return nil, err
		}

		teamAccessesMap, err := parseAccessMap(teamAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid team accesses: %w", err)
		}

		err = s.cli.UpdateAccessGroupTeamAccesses(id, teamAccessesMap)
		if err != nil {
			return nil, fmt.Errorf("failed to update access group team accesses: %w", err)
		}

		return mcp.NewToolResultText("Access group team accesses updated successfully"), nil
	}
}

func (s *PortainerMCPServer) handleAddEnvironmentToAccessGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		environmentId, err := parser.GetInt("environmentId", true)
		if err != nil {
			return nil, err
		}

		err = s.cli.AddEnvironmentToAccessGroup(id, environmentId)
		if err != nil {
			return nil, fmt.Errorf("failed to add environment to access group: %w", err)
		}

		return mcp.NewToolResultText("Environment added to access group successfully"), nil
	}
}

func (s *PortainerMCPServer) handleRemoveEnvironmentFromAccessGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		environmentId, err := parser.GetInt("environmentId", true)
		if err != nil {
			return nil, err
		}

		err = s.cli.RemoveEnvironmentFromAccessGroup(id, environmentId)
		if err != nil {
			return nil, fmt.Errorf("failed to remove environment from access group: %w", err)
		}

		return mcp.NewToolResultText("Environment removed from access group successfully"), nil
	}
}
