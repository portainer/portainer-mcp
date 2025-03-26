package mcp

import (
	"context"
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/mark3labs/mcp-go/mcp"
)

// CreateAccessGroupsResourceHandler creates a handler for getting access groups
func CreateAccessGroupsResourceHandler(s *PortainerMCPServer) *ResourceHandler {
	return NewResourceHandler(
		ResourceURIAccessGroups,
		"Portainer Access Groups",
		"Lists all available access groups",
		CreateResourceHandler("Access Groups", "Lists all available access groups", ResourceURIAccessGroups,
			func(ctx context.Context, request mcp.ReadResourceRequest) (interface{}, error) {
				s.Debug("Handling get access groups request")
				accessGroups, err := s.cli.GetAccessGroups()
				if err != nil {
					return nil, NewClientError("failed to get access groups", err)
				}
				return accessGroups, nil
			},
		),
	)
}

// CreateCreateAccessGroupToolHandler creates a handler for creating an access group
func CreateCreateAccessGroupToolHandler(s *PortainerMCPServer) *ToolHandler {
	// Create the tool definition directly
	createAccessGroupTool := mcp.NewTool("createAccessGroup",
		mcp.WithDescription("Create a new access group. Use this tool when you want to define accesses on more than one environment. "+
			"Otherwise, define the accesses on the environment level."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the access group"),
		),
		mcp.WithArray("environmentIds",
			mcp.Description("The IDs of the environments that are part of the access group. "+
				"Must include all the environment IDs that are part of the group - this includes new environments and the existing environments that are already associated with the group. "+
				"Example: [1, 2, 3]."),
			mcp.Items(map[string]any{
				"type": "number",
			}),
		),
		mcp.WithArray("userAccesses",
			mcp.Description("The user accesses that are associated with all the environments in the access group. "+
				"The ID is the user ID of the user in Portainer. "+
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
			mcp.Description("The team accesses that are associated with all the environments in the access group. "+
				"The ID is the team ID of the team in Portainer. "+
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
		s.Debug("Handling create access group request")
		parser := NewParameterParser(request)

		// Parse required parameters
		name, err := parser.GetString("name", true)
		if err != nil {
			return nil, err
		}

		// Parse optional parameters
		environmentIds, err := parser.GetNumericArray("environmentIds", false)
		if err != nil {
			return nil, err
		}

		userAccesses, err := parser.GetAccessMap("userAccesses", false)
		if err != nil {
			return nil, err
		}

		teamAccesses, err := parser.GetAccessMap("teamAccesses", false)
		if err != nil {
			return nil, err
		}

		// Create access group model
		accessGroup := models.AccessGroup{
			Name:           name,
			EnvironmentIds: environmentIds,
			UserAccesses:   userAccesses,
			TeamAccesses:   teamAccesses,
		}

		// Call client
		groupID, err := s.cli.CreateAccessGroup(accessGroup)
		if err != nil {
			return nil, NewClientError("failed to create access group", err)
		}

		return CreateSuccessResponse(fmt.Sprintf("Access group created successfully with ID: %d", groupID)), nil
	}

	return NewToolHandler(createAccessGroupTool, handlerFunc)
}

// CreateUpdateAccessGroupToolHandler creates a handler for updating an access group
func CreateUpdateAccessGroupToolHandler(s *PortainerMCPServer) *ToolHandler {
	// Create the tool definition directly
	updateAccessGroupTool := mcp.NewTool("updateAccessGroup",
		mcp.WithDescription("Update an existing access group."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the access group to update"),
		),
		mcp.WithString("name",
			mcp.Description("The name of the access group, re-use the existing name to keep the same group name"),
		),
		mcp.WithArray("userAccesses",
			mcp.Description("The user accesses that are associated with all the environments in the access group. "+
				"The ID is the user ID of the user in Portainer. "+
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
			mcp.Description("The team accesses that are associated with all the environments in the access group. "+
				"The ID is the team ID of the team in Portainer. "+
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
			s.Debug("Handling update access group request")
			parser := NewParameterParser(request)

			// Parse required parameters
			id, err := parser.GetInt("id", true)
			if err != nil {
				return nil, err
			}

			// Parse optional parameters
			name, err := parser.GetString("name", false)
			if err != nil {
				return nil, err
			}

			userAccesses, err := parser.GetAccessMap("userAccesses", false)
			if err != nil {
				return nil, err
			}

			teamAccesses, err := parser.GetAccessMap("teamAccesses", false)
			if err != nil {
				return nil, err
			}

			// Create access group model
			accessGroup := models.AccessGroup{
				ID:           id,
				Name:         name,
				UserAccesses: userAccesses,
				TeamAccesses: teamAccesses,
			}

			// Call client
			err = s.cli.UpdateAccessGroup(accessGroup)
			if err != nil {
				return nil, NewClientError("failed to update access group", err)
			}

			return CreateSuccessResponse("Access group updated successfully"), nil
		}

	return NewToolHandler(updateAccessGroupTool, handlerFunc)
}

// CreateAddEnvironmentToAccessGroupToolHandler creates a handler for adding an environment to an access group
func CreateAddEnvironmentToAccessGroupToolHandler(s *PortainerMCPServer) *ToolHandler {
	// Create the tool definition directly
	addEnvToAccessGroupTool := mcp.NewTool("addEnvironmentToAccessGroup",
		mcp.WithDescription("Add an environment to an access group."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the access group to update"),
		),
		mcp.WithNumber("environmentId",
			mcp.Required(),
			mcp.Description("The ID of the environment to add to the access group"),
		),
	)

	// Create the handler function
	handlerFunc := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.Debug("Handling add environment to access group request")
		parser := NewParameterParser(request)

		// Parse required parameters
		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		environmentId, err := parser.GetInt("environmentId", true)
		if err != nil {
			return nil, err
		}

		// Call client
		err = s.cli.AddEnvironmentToAccessGroup(id, environmentId)
		if err != nil {
			return nil, NewClientError("failed to add environment to access group", err)
		}

		return CreateSuccessResponse(fmt.Sprintf("Environment %d added to access group %d successfully", environmentId, id)), nil
	}

	return NewToolHandler(addEnvToAccessGroupTool, handlerFunc)
}

// CreateRemoveEnvironmentFromAccessGroupToolHandler creates a handler for removing an environment from an access group
func CreateRemoveEnvironmentFromAccessGroupToolHandler(s *PortainerMCPServer) *ToolHandler {
	// Create the tool definition directly
	removeEnvFromAccessGroupTool := mcp.NewTool("removeEnvironmentFromAccessGroup",
		mcp.WithDescription("Remove an environment from an access group."),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the access group to update"),
		),
		mcp.WithNumber("environmentId",
			mcp.Required(),
			mcp.Description("The ID of the environment to remove from the access group"),
		),
	)

	// Create the handler function
	handlerFunc := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			s.Debug("Handling remove environment from access group request")
			parser := NewParameterParser(request)

			// Parse required parameters
			id, err := parser.GetInt("id", true)
			if err != nil {
				return nil, err
			}

			environmentId, err := parser.GetInt("environmentId", true)
			if err != nil {
				return nil, err
			}

			// Call client
			err = s.cli.RemoveEnvironmentFromAccessGroup(id, environmentId)
			if err != nil {
				return nil, NewClientError("failed to remove environment from access group", err)
			}

			return CreateSuccessResponse(fmt.Sprintf("Environment %d removed from access group %d successfully", environmentId, id)), nil
		}

	return NewToolHandler(removeEnvFromAccessGroupTool, handlerFunc)
}