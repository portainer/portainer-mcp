package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

func (s *PortainerMCPServer) AddTeamFeatures() {
	s.addToolIfExists(ToolListTeams, s.HandleGetTeams())

	if !s.readOnly {
		s.addToolIfExists(ToolCreateTeam, s.HandleCreateTeam())
		s.addToolIfExists(ToolUpdateTeamName, s.HandleUpdateTeamName())
		s.addToolIfExists(ToolUpdateTeamMembers, s.HandleUpdateTeamMembers())
	}
}

func (s *PortainerMCPServer) HandleCreateTeam() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleGetTeams() server.ToolHandlerFunc {
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

func (s *PortainerMCPServer) HandleUpdateTeamName() server.ToolHandlerFunc {
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

		err = s.cli.UpdateTeamName(id, name)
		if err != nil {
			return nil, fmt.Errorf("failed to update team. Error: %w", err)
		}

		return mcp.NewToolResultText("Team updated successfully"), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateTeamMembers() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		userIds, err := parser.GetArrayOfIntegers("userIds", true)
		if err != nil {
			return nil, err
		}

		err = s.cli.UpdateTeamMembers(id, userIds)
		if err != nil {
			return nil, fmt.Errorf("failed to update team members. Error: %w", err)
		}

		return mcp.NewToolResultText("Team members updated successfully"), nil
	}
}
