package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/toolgen"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddTeamFeatures() {
	createTeamTool := s.tools[ToolCreateTeam]
	s.srv.AddTool(createTeamTool, s.handleCreateTeam())

	// listTeamsTool := s.tools[ToolListTeams]
	// s.srv.AddTool(listTeamsTool, s.handleGetTeams())

	updateTeamTool := s.tools[ToolUpdateTeam]
	s.srv.AddTool(updateTeamTool, s.handleUpdateTeam())
}

func (s *PortainerMCPServer) handleCreateTeam() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		name, err := parser.GetString("name", true)
		if err != nil {
			return nil, err
		}

		id, err := s.cli.CreateTeam(name)
		if err != nil {
			return nil, fmt.Errorf("failed to create team: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Team created successfully with ID: %d", id)), nil
	}
}

func (s *PortainerMCPServer) handleGetTeams() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		teams, err := s.cli.GetTeams()
		if err != nil {
			return nil, fmt.Errorf("failed to get teams: %w", err)
		}

		data, err := json.Marshal(teams)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal teams: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) handleUpdateTeam() server.ToolHandlerFunc {
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

		userIds, err := parser.GetArrayOfIntegers("userIds", false)
		if err != nil {
			return nil, err
		}

		if name == "" && len(userIds) == 0 {
			return nil, fmt.Errorf("team name or user IDs are required")
		}

		if name != "" {
			err := s.cli.UpdateTeam(id, name)
			if err != nil {
				return nil, fmt.Errorf("failed to update team. Error: %w", err)
			}
		}

		if len(userIds) > 0 {
			err = s.cli.UpdateTeamMembers(id, userIds)
			if err != nil {
				return nil, fmt.Errorf("failed to update team members. Error: %w", err)
			}
		}

		return mcp.NewToolResultText("Team updated successfully"), nil
	}
}
