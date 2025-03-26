package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// CreateSettingsResourceHandler creates a handler for getting settings
func CreateSettingsResourceHandler(s *PortainerMCPServer) *ResourceHandler {
	return NewResourceHandler(
		ResourceURISettings,
		"Portainer Settings",
		"Get Portainer settings",
		CreateResourceHandler("Settings", "Get Portainer settings", ResourceURISettings,
			func(ctx context.Context, request mcp.ReadResourceRequest) (interface{}, error) {
				s.Debug("Handling get settings request")
				// Stub implementation - real implementation would call s.cli.GetSettings()
				return map[string]interface{}{
					"version": "2.19.0",
					"logoURL": "",
					"authentication": true,
					"analytics": false,
				}, nil
			},
		),
	)
}