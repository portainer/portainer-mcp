package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddEnvironmentFeatures() {
	environmentsResource := mcp.NewResource("portainer://environments",
		"Portainer Environments",
		mcp.WithResourceDescription("Lists all available environments"),
		mcp.WithMIMEType("application/json"),
	)

	updateEnvironmentTool := mcp.NewTool("updateEnvironment",
		mcp.WithDescription("Update an existing environment"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the environment to update"),
		),
		mcp.WithString("tagIds",
			mcp.Required(),
			mcp.Description("The IDs of the tags that are associated with the environment, separated by commas."+
				"Must include all the tag IDs that are associated with the environment - this includes new tags and the existing tags that are already associated with the environment."),
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
						"enum":        []string{"environment_administrator", "helpdesk_user", "standard_user", "readonly_user", "operator_user"},
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
						"enum":        []string{"environment_administrator", "helpdesk_user", "standard_user", "readonly_user", "operator_user"},
					},
				},
			}),
		),
	)

	s.srv.AddResource(environmentsResource, s.handleGetEnvironments())
	s.srv.AddTool(updateEnvironmentTool, s.handleUpdateEnvironment())
}

func (s *PortainerMCPServer) handleGetEnvironments() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		environments, err := s.cli.GetEnvironments()
		if err != nil {
			return nil, fmt.Errorf("failed to get environments: %w", err)
		}

		data, err := json.Marshal(environments)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environments: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "portainer://environments",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	}
}

func (s *PortainerMCPServer) handleUpdateEnvironment() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := request.Params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("environment ID is required")
		}

		tagIds, ok := request.Params.Arguments["tagIds"].(string)
		if !ok {
			return nil, fmt.Errorf("tag IDs are required")
		}

		tagIdsInt, err := parseCommaSeparatedInts(tagIds)
		if err != nil {
			return nil, fmt.Errorf("invalid tag IDs. Error: %w", err)
		}

		// Parse optional user accesses
		userAccessesMap := map[int]string{}
		if userAccesses, ok := request.Params.Arguments["userAccesses"].([]any); ok {
			var err error
			userAccessesMap, err = parseAccessMap(userAccesses)
			if err != nil {
				return nil, fmt.Errorf("invalid user accesses: %w", err)
			}
		}

		// Parse optional team accesses
		teamAccessesMap := map[int]string{}
		if teamAccesses, ok := request.Params.Arguments["teamAccesses"].([]any); ok {
			var err error
			teamAccessesMap, err = parseAccessMap(teamAccesses)
			if err != nil {
				return nil, fmt.Errorf("invalid team accesses: %w", err)
			}
		}

		err = s.cli.UpdateEnvironment(int(id), tagIdsInt, userAccessesMap, teamAccessesMap)
		if err != nil {
			return nil, fmt.Errorf("error updating environment. Error: %w", err)
		}

		return mcp.NewToolResultText("Environment updated successfully"), nil
	}
}
