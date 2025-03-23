package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddUserFeatures() {
	usersResource := mcp.NewResource("portainer://users",
		"Portainer Users",
		mcp.WithResourceDescription("Lists all available users"),
		mcp.WithMIMEType("application/json"),
	)

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

	s.srv.AddResource(usersResource, s.handleGetUsers())
	s.srv.AddTool(updateUserTool, s.handleUpdateUser())
}

func (s *PortainerMCPServer) handleGetUsers() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		users, err := s.cli.GetUsers()
		if err != nil {
			return nil, fmt.Errorf("failed to get users: %w", err)
		}

		data, err := json.Marshal(users)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal users: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "portainer://users",
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	}
}

func (s *PortainerMCPServer) handleUpdateUser() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := request.Params.Arguments["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("user ID is required")
		}

		role, ok := request.Params.Arguments["role"].(string)
		if !ok {
			return nil, fmt.Errorf("role is required")
		}

		if role != "admin" && role != "user" && role != "edge_admin" {
			return nil, fmt.Errorf("invalid role: must be admin, user or edge_admin")
		}

		err := s.cli.UpdateUser(int(id), role)
		if err != nil {
			return nil, fmt.Errorf("error updating user. Error: %w", err)
		}

		return mcp.NewToolResultText("User updated successfully"), nil
	}
}
