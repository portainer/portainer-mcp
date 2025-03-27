package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/pkg/toolgen"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddAccessGroupFeatures() {
	s.addToolIfExists(ToolCreateAccessGroup, s.handleCreateAccessGroup())
	s.addToolIfExists(ToolListAccessGroups, s.handleGetAccessGroups())
	s.addToolIfExists(ToolUpdateAccessGroup, s.handleUpdateAccessGroup())
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

		userAccesses, err := parser.GetArrayOfObjects("userAccesses", false)
		if err != nil {
			return nil, err
		}
		userAccessesMap, err := parseAccessMap(userAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid user accesses: %w", err)
		}

		teamAccesses, err := parser.GetArrayOfObjects("teamAccesses", false)
		if err != nil {
			return nil, err
		}
		teamAccessesMap, err := parseAccessMap(teamAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid team accesses: %w", err)
		}

		// Create access group
		accessGroup := models.AccessGroup{
			Name:           name,
			EnvironmentIds: environmentIds,
			UserAccesses:   userAccessesMap,
			TeamAccesses:   teamAccessesMap,
		}

		groupID, err := s.cli.CreateAccessGroup(accessGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to create access group: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Access group created successfully with ID: %d", groupID)), nil
	}
}

func (s *PortainerMCPServer) handleUpdateAccessGroup() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		name, err := parser.GetString("name", false)
		if err != nil {
			return nil, err
		}

		userAccesses, err := parser.GetArrayOfObjects("userAccesses", false)
		if err != nil {
			return nil, err
		}
		userAccessesMap, err := parseAccessMap(userAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid user accesses: %w", err)
		}

		teamAccesses, err := parser.GetArrayOfObjects("teamAccesses", false)
		if err != nil {
			return nil, err
		}
		teamAccessesMap, err := parseAccessMap(teamAccesses)
		if err != nil {
			return nil, fmt.Errorf("invalid team accesses: %w", err)
		}

		accessGroup := models.AccessGroup{
			ID:           id,
			Name:         name,
			UserAccesses: userAccessesMap,
			TeamAccesses: teamAccessesMap,
		}

		err = s.cli.UpdateAccessGroup(accessGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to update access group: %w", err)
		}

		return mcp.NewToolResultText("Access group updated successfully"), nil
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
