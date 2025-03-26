package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// CreateEnvironmentsResourceHandler creates a handler for getting environments
func CreateEnvironmentsResourceHandler(s *PortainerMCPServer) *ResourceHandler {
	return NewResourceHandler(
		ResourceURIEnvironments,
		"Portainer Environments",
		"Lists all available environments",
		CreateResourceHandler("Environments", "Lists all available environments", ResourceURIEnvironments,
			func(ctx context.Context, request mcp.ReadResourceRequest) (interface{}, error) {
				s.Debug("Handling get environments request")
				environments, err := s.cli.GetEnvironments()
				if err != nil {
					return nil, NewClientError("failed to get environments", err)
				}
				return environments, nil
			},
		),
	)
}

// CreateUpdateEnvironmentToolHandler creates a handler for updating an environment
func CreateUpdateEnvironmentToolHandler(s *PortainerMCPServer) *ToolHandler {
	// Create the tool definition directly
	updateEnvTool := mcp.NewTool("updateEnvironment",
		mcp.WithDescription("Update an existing environment"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the environment to update"),
		),
		mcp.WithArray("tagIds",
			mcp.Required(),
			mcp.Description("The IDs of the tags that are associated with the environment."+
				"Must include all the tag IDs that are associated with the environment - this includes new tags and the existing tags that are already associated with the environment."+
				"Example: [1, 2, 3]."),
			mcp.Items(map[string]any{
				"type": "number",
			}),
		),
		mcp.WithArray("userAccesses",
			mcp.Description("The user accesses that are associated with all the environments in the access group."+
				"The ID is the user ID of the user in Portainer."+
				"Must include all the access policies for all the users that are associated with the environment - this includes new users and the existing users that are already associated with the environment."+
				"Example: [{id: 1, access: 'environment_administrator'}, {id: 2, access: 'standard_user'}]."),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "number",
						"description": "The ID of the user",
					},
					"access": map[string]any{
						"type":        "string",
						"description": "The access level of the user. Can be environment_administrator, helpdesk_user, standard_user, readonly_user or operator_user",
						"enum":        AllAccessLevels,
					},
				},
			}),
		),
		mcp.WithArray("teamAccesses",
			mcp.Description("The team accesses that are associated with all the environments in the access group."+
				"The ID is the team ID of the team in Portainer."+
				"Must include all the access policies for all the teams that are associated with the environment - this includes new teams and the existing teams that are already associated with the environment."+
				"Example: [{id: 1, access: 'environment_administrator'}, {id: 2, access: 'standard_user'}]."),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "number",
						"description": "The ID of the team",
					},
					"access": map[string]any{
						"type":        "string",
						"description": "The access level of the team. Can be environment_administrator, helpdesk_user, standard_user, readonly_user or operator_user",
						"enum":        AllAccessLevels,
					},
				},
			}),
		),
	)

	// Create the handler function
	handlerFunc := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.Debug("Handling update environment request")
		parser := NewParameterParser(request)

		// Parse required parameters
		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		tagIds, err := parser.GetNumericArray("tagIds", true)
		if err != nil {
			return nil, err
		}

		// Parse optional parameters
		userAccesses, err := parser.GetAccessMap("userAccesses", false)
		if err != nil {
			return nil, err
		}

		teamAccesses, err := parser.GetAccessMap("teamAccesses", false)
		if err != nil {
			return nil, err
		}

		// Call client
		err = s.cli.UpdateEnvironment(id, tagIds, userAccesses, teamAccesses)
		if err != nil {
			return nil, NewClientError("error updating environment", err)
		}

		return CreateSuccessResponse(fmt.Sprintf("Environment %d updated successfully", id)), nil
	}

	return NewToolHandler(updateEnvTool, handlerFunc)
}