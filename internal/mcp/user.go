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
			return mcp.NewToolResultErrorFromErr("failed to get users", err), nil
		}

		data, err := json.Marshal(users)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal users", err), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func (s *PortainerMCPServer) HandleUpdateUserRole() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parser := toolgen.NewParameterParser(request)

		id, err := parser.GetInt("id", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid id parameter", err), nil
		}

		role, err := parser.GetString("role", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid role parameter", err), nil
		}

		if !isValidUserRole(role) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid role %s: must be one of: %v", role, AllUserRoles)), nil
		}

		err = s.cli.UpdateUserRole(id, role)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to update user role", err), nil
		}

		return mcp.NewToolResultText("User updated successfully"), nil
	}
}
