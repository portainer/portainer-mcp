package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/toolgen"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddUserFeatures() {
	usersResource := mcp.NewResource("portainer://users",
		"Portainer Users",
		mcp.WithResourceDescription("Lists all available users"),
		mcp.WithMIMEType("application/json"),
	)

	s.srv.AddResource(usersResource, s.handleGetUsers())

	updateUserTool := s.tools[ToolUpdateUser]
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
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return nil, err
		}

		role, err := parser.GetString("role", true)
		if err != nil {
			return nil, err
		}

		if !isValidUserRole(role) {
			return nil, fmt.Errorf("invalid role %s: must be one of: %v", role, AllUserRoles)
		}

		err = s.cli.UpdateUser(id, role)
		if err != nil {
			return nil, fmt.Errorf("error updating user. Error: %w", err)
		}

		return mcp.NewToolResultText("User updated successfully"), nil
	}
}
