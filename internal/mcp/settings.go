package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *PortainerMCPServer) AddSettingsFeatures() {
	settingsResource := mcp.NewResource("portainer://settings",
		"Portainer Settings",
		mcp.WithResourceDescription("Inspect Portainer instance settings"),
		mcp.WithMIMEType("application/json"),
	)

	s.srv.AddResource(settingsResource, s.handleGetSettings())
}

func (s *PortainerMCPServer) handleGetSettings() server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]interface{}, error) {
		settings, err := s.cli.GetSettings()
		if err != nil {
			return nil, fmt.Errorf("failed to get settings: %w", err)
		}

		data, err := json.Marshal(settings)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal settings: %w", err)
		}

		return []interface{}{
			mcp.TextResourceContents{
				ResourceContents: mcp.ResourceContents{
					URI:      "portainer://settings",
					MIMEType: "application/json",
				},
				Text: string(data),
			},
		}, nil
	}
}
