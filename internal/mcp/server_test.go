package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPortainerMCPServer(t *testing.T) {
	tests := []struct {
		name          string
		serverURL     string
		token         string
		toolsPath     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "successful initialization",
			serverURL:   "https://portainer.example.com",
			token:       "valid-token",
			toolsPath:   "testdata/valid_tools.yaml",
			expectError: false,
		},
		{
			name:          "invalid tools path",
			serverURL:     "https://portainer.example.com",
			token:         "valid-token",
			toolsPath:     "testdata/nonexistent.yaml",
			expectError:   true,
			errorContains: "failed to load tools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewPortainerMCPServer(tt.serverURL, tt.token, tt.toolsPath)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.ErrorContains(t, err, tt.errorContains)
				}
				assert.Nil(t, server)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, server)
				assert.NotNil(t, server.srv)
				assert.NotNil(t, server.cli)
				assert.NotNil(t, server.tools)
			}
		})
	}
}

func TestAddToolIfExists(t *testing.T) {
	tests := []struct {
		name     string
		tools    map[string]mcp.Tool
		toolName string
		exists   bool
	}{
		{
			name: "existing tool",
			tools: map[string]mcp.Tool{
				"test_tool": {
					Name:        "test_tool",
					Description: "Test tool description",
					InputSchema: mcp.ToolInputSchema{
						Properties: map[string]any{},
					},
				},
			},
			toolName: "test_tool",
			exists:   true,
		},
		{
			name: "non-existing tool",
			tools: map[string]mcp.Tool{
				"test_tool": {
					Name:        "test_tool",
					Description: "Test tool description",
					InputSchema: mcp.ToolInputSchema{
						Properties: map[string]any{},
					},
				},
			},
			toolName: "nonexistent_tool",
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create server with test tools
			mcpServer := server.NewMCPServer(
				"Test Server",
				"1.0.0",
				server.WithResourceCapabilities(true, true),
				server.WithLogging(),
			)
			server := &PortainerMCPServer{
				tools: tt.tools,
				srv:   mcpServer,
			}

			// Create a handler function
			handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return &mcp.CallToolResult{}, nil
			}

			// Call addToolIfExists
			server.addToolIfExists(tt.toolName, handler)

			// Verify if the tool exists in the tools map
			_, toolExists := server.tools[tt.toolName]
			assert.Equal(t, tt.exists, toolExists)
		})
	}
}
