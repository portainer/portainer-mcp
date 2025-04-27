package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddSettingsFeatures() {
	s.addToolIfExists(ToolGetSettings, s.HandleGetSettings())
}

func (s *PortainerMCPServer) HandleGetSettings() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		settings, err := s.cli.GetSettings()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to get settings", err), nil
		}

		data, err := json.Marshal(settings)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal settings", err), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}
