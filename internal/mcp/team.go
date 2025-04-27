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
			return mcp.NewToolResultErrorFromErr("invalid name parameter", err), nil
		}

		teamID, err := s.cli.CreateTeam(name)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create team", err), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Team created successfully with ID: %d", teamID)), nil
	}
}

func (s *PortainerMCPServer) HandleGetTeams() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		teams, err := s.cli.GetTeams()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get teams", err), nil
		}

		data, err := json.Marshal(teams)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal teams", err), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateTeamName() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		name, err := parser.GetString("name", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid name parameter", err), nil
		}

		err = s.cli.UpdateTeamName(id, name)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update team name", err), nil
		}

		return mcp.NewToolResultText("Team name updated successfully"), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateTeamMembers() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		userIDs, err := parser.GetArrayOfIntegers("userIds", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid userIds parameter", err), nil
		}

		err = s.cli.UpdateTeamMembers(id, userIDs)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update team members", err), nil
		}

		return mcp.NewToolResultText("Team members updated successfully"), nil
	}
}
