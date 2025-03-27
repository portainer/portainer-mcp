package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddSettingsFeatures() {
	s.addToolIfExists(ToolGetSettings, s.handleGetSettings())
}

func (s *PortainerMCPServer) handleGetSettings() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		settings, err := s.cli.GetSettings()
		if err != nil {
			return nil, fmt.Errorf("failed to get settings: %w", err)
		}

		data, err := json.Marshal(settings)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal settings: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}
