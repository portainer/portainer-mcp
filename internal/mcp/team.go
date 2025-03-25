package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddTeamFeatures() {
	teamsResource := mcp.NewResource("portainer://teams",
		"Portainer Teams",
		mcp.WithResourceDescription("Lists all available teams"),
		mcp.WithMIMEType("application/json"),
	)

	createTeamTool := mcp.NewTool("createTeam",
		mcp.WithDescription("Create a new team"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the team"),
		),
	)

	updateTeamTool := mcp.NewTool("updateTeam",
		mcp.WithDescription("Update an existing team"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the team to update"),
		),
		mcp.WithString("name",
			mcp.Description("The new name of the team"),
		),
		mcp.WithString("userIds",
			mcp.Description("The IDs of the users that are part of the team, separated by commas."+
				"Must include all the user IDs that are part of the team - this includes new users and the existing users that are already associated with the team."),
		),
	)

	s.srv.AddResource(teamsResource, s.handleGetTeams())
	s.srv.AddTool(createTeamTool, s.handleCreateTeam())
	s.srv.AddTool(updateTeamTool, s.handleUpdateTeam())
}

func (s *PortainerMCPServer) handleCreateTeam() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.Params.Arguments["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("team name is required")
		}

		id, err := s.cli.CreateTeam(name)
		if err != nil {
			return nil, fmt.Errorf("failed to create team: %w", err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("Team created successfully with ID: %d", id)), nil
	}
}

func (s *PortainerMCPServer) handleGetTeams() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		teams, err := s.cli.GetTeams()
		if err != nil {
			return nil, fmt.Errorf("failed to get teams: %w", err)
		}

		data, err := json.Marshal(teams)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal teams: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "portainer://teams",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	}
}

func (s *PortainerMCPServer) handleUpdateTeam() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := request.Params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("team ID is required")
		}

		name := request.Params.Arguments["name"].(string)
		userIds := request.Params.Arguments["userIds"].(string)
		if name == "" && userIds == "" {
			return nil, fmt.Errorf("team name or user IDs are required")
		}

		if name != "" {
			err := s.cli.UpdateTeam(int(id), name)
			if err != nil {
				return nil, fmt.Errorf("failed to update team. Error: %w", err)
			}
		}

		if userIds != "" {
			userIdsList, err := parseCommaSeparatedInts(userIds)
			if err != nil {
				return nil, fmt.Errorf("invalid user IDs. Error: %w", err)
			}

			err = s.cli.UpdateTeamMembers(int(id), userIdsList)
			if err != nil {
				return nil, fmt.Errorf("failed to update team members. Error: %w", err)
			}
		}

		return mcp.NewToolResultText("Team updated successfully"), nil
	}
}
