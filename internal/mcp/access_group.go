package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

func (s *PortainerMCPServer) AddAccessGroupFeatures() {
	s.addToolIfExists(ToolCreateAccessGroup, s.HandleCreateAccessGroup())
	s.addToolIfExists(ToolListAccessGroups, s.HandleGetAccessGroups())
	s.addToolIfExists(ToolUpdateAccessGroupName, s.HandleUpdateAccessGroupName())
	s.addToolIfExists(ToolUpdateAccessGroupUserAccesses, s.HandleUpdateAccessGroupUserAccesses())
	s.addToolIfExists(ToolUpdateAccessGroupTeamAccesses, s.HandleUpdateAccessGroupTeamAccesses())
	s.addToolIfExists(ToolAddEnvironmentToAccessGroup, s.HandleAddEnvironmentToAccessGroup())
	s.addToolIfExists(ToolRemoveEnvironmentFromAccessGroup, s.HandleRemoveEnvironmentFromAccessGroup())
}

func (s *PortainerMCPServer) HandleGetAccessGroups() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleCreateAccessGroup() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleUpdateAccessGroupName() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleUpdateAccessGroupUserAccesses() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleUpdateAccessGroupTeamAccesses() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleAddEnvironmentToAccessGroup() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleRemoveEnvironmentFromAccessGroup() server.ToolHandlerFunc {
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
