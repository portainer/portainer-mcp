package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/toolgen"
)

func (s *PortainerMCPServer) AddUserFeatures() {
	s.addToolIfExists(ToolListUsers, s.HandleGetUsers())

	if !s.readOnly {
		s.addToolIfExists(ToolUpdateUserRole, s.HandleUpdateUserRole())
	}
}

func (s *PortainerMCPServer) HandleGetUsers() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		users, err := s.cli.GetUsers()
		if err != nil {
			return nil, fmt.Errorf("failed to get users: %w", err)
		}

		data, err := json.Marshal(users)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal users: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateUserRole() server.ToolHandlerFunc {
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

		err = s.cli.UpdateUserRole(id, role)
		if err != nil {
			return nil, fmt.Errorf("error updating user. Error: %w", err)
		}

		return mcp.NewToolResultText("User updated successfully"), nil
	}
}
