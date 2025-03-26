package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// CreateUsersResourceHandler creates a handler for getting users
func CreateUsersResourceHandler(s *PortainerMCPServer) *ResourceHandler {
	return NewResourceHandler(
		ResourceURIUsers,
		"Portainer Users",
		"Lists all available users",
		CreateResourceHandler("Users", "Lists all available users", ResourceURIUsers,
			func(ctx context.Context, request mcp.ReadResourceRequest) (interface{}, error) {
				s.Debug("Handling get users request")
				users, err := s.cli.GetUsers()
				if err != nil {
					return nil, NewClientError("failed to get users", err)
				}
				return users, nil
			},
		),
	)
}

// CreateUpdateUserToolHandler creates a handler for updating a user
func CreateUpdateUserToolHandler(s *PortainerMCPServer) *ToolHandler {
	// Create the tool definition directly
	updateUserTool := mcp.NewTool("updateUser",
		mcp.WithDescription("Update an existing user"),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("The ID of the user to update"),
		),
		mcp.WithString("role",
			mcp.Required(),
			mcp.Description("The role of the user. Can be admin, user or edge_admin"),
			mcp.Enum("admin", "user", "edge_admin"),
		),
	)

	// Create the handler function
	handlerFunc := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.Debug("Handling update user request")
		parser := NewParameterParser(request)

		// Parse required parameters
		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		role, err := parser.GetString("role", true)
		if err != nil {
			return nil, err
		}

		if !IsValidUserRole(role) {
			return nil, NewInvalidParameterError(
				fmt.Sprintf("invalid role: %s. Must be one of: %v", role, AllUserRoles),
				nil,
			)
		}

		// Call client
		err = s.cli.UpdateUser(id, role)
		if err != nil {
			return nil, NewClientError("error updating user", err)
		}

		return CreateSuccessResponse(fmt.Sprintf("User %d updated successfully with role %s", id, role)), nil
	}

	return NewToolHandler(updateUserTool, handlerFunc)
}