package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/stretchr/testify/assert"
)

func TestHandleGetSettings(t *testing.T) {
	tests := []struct {
		name          string
		settings      models.PortainerSettings
		mockError     error
		expectError   bool
		errorContains string
	}{
		{
			name: "successful settings retrieval",
			settings: models.PortainerSettings{
				Authentication: struct {
					Method string `json:"method"`
				}{
					Method: models.AuthenticationMethodInternal,
				},
				Edge: struct {
					Enabled   bool   `json:"enabled"`
					ServerURL string `json:"server_url"`
				}{
					Enabled:   true,
					ServerURL: "https://example.com",
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:          "client error",
			settings:      models.PortainerSettings{},
			mockError:     assert.AnError,
			expectError:   true,
			errorContains: "failed to get settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := new(MockPortainerClient)
			mockClient.On("GetSettings").Return(tt.settings, tt.mockError)

			// Create server with mock client
			srv := &PortainerMCPServer{
				srv:   server.NewMCPServer("Test Server", "1.0.0"),
				cli:   mockClient,
				tools: make(map[string]mcp.Tool),
			}

			// Get the handler
			handler := srv.HandleGetSettings()

			// Call the handler
			result, err := handler(context.Background(), mcp.CallToolRequest{})

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.ErrorContains(t, err, tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)

				var settings models.PortainerSettings
				err = json.Unmarshal([]byte(textContent.Text), &settings)
				assert.NoError(t, err)
				assert.Equal(t, tt.settings, settings)
			}

			// Verify mock expectations
			mockClient.AssertExpectations(t)
		})
	}
}
