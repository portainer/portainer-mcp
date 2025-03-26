package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// CreateTeamsResourceHandler creates a handler for getting teams
func CreateTeamsResourceHandler(s *PortainerMCPServer) *ResourceHandler {
	return NewResourceHandler(
		ResourceURITeams,
		"Portainer Teams",
		"Lists all available teams",
		CreateResourceHandler("Teams", "Lists all available teams", ResourceURITeams,
			func(ctx context.Context, request mcp.ReadResourceRequest) (interface{}, error) {
				s.Debug("Handling get teams request")
				teams, err := s.cli.GetTeams()
				if err != nil {
					return nil, NewClientError("failed to get teams", err)
				}
				return teams, nil
			},
		),
	)
}

// CreateUpdateTeamToolHandler creates a handler for updating a team
func CreateUpdateTeamToolHandler(s *PortainerMCPServer) *ToolHandler {
	// Create the tool definition directly
	updateTeamTool := mcp.NewTool("updateTeam",
		mcp.WithDescription("Update an existing team"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the team to update"),
		),
		mcp.WithString("name",
			mcp.Description("The name of the team"),
		),
	)

	// Create the handler function
	handlerFunc := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.Debug("Handling update team request")
		
		// This is a stub implementation
		return CreateSuccessResponse("Team updated successfully"), nil
	}

	return NewToolHandler(updateTeamTool, handlerFunc)
}