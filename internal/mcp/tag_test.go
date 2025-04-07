package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestHandleGetEnvironmentTags(t *testing.T) {
	tests := []struct {
		name         string
		mockTags     []models.EnvironmentTag
		mockError    error
		expectError  bool
		expectedJSON string
	}{
		{
			name: "successful tags retrieval",
			mockTags: []models.EnvironmentTag{
				{ID: 1, Name: "tag1"},
				{ID: 2, Name: "tag2"},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "api error",
			mockTags:    nil,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockPortainerClient{}
			mockClient.On("GetEnvironmentTags").Return(tt.mockTags, tt.mockError)

			// Create server with mock client
			server := &PortainerMCPServer{
				cli: mockClient,
			}

			// Call handler
			handler := server.HandleGetEnvironmentTags()
			result, err := handler(context.Background(), mcp.CallToolRequest{})

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				if tt.mockError != nil {
					assert.ErrorContains(t, err, tt.mockError.Error())
				}
			} else {
				assert.NoError(t, err)

				// Verify JSON response
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)

				var tags []models.EnvironmentTag
				err = json.Unmarshal([]byte(textContent.Text), &tags)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockTags, tags)
			}

			// Verify mock expectations
			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleCreateEnvironmentTag(t *testing.T) {
	tests := []struct {
		name        string
		inputName   string
		mockID      int
		mockError   error
		expectError bool
	}{
		{
			name:        "successful tag creation",
			inputName:   "test-tag",
			mockID:      123,
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "api error",
			inputName:   "test-tag",
			mockID:      0,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
		},
		{
			name:        "missing name parameter",
			inputName:   "",
			mockID:      0,
			mockError:   nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockPortainerClient{}
			if tt.inputName != "" {
				mockClient.On("CreateEnvironmentTag", tt.inputName).Return(tt.mockID, tt.mockError)
			}

			// Create server with mock client
			server := &PortainerMCPServer{
				cli: mockClient,
			}

			// Create request with parameters
			request := CreateMCPRequest(map[string]any{})
			if tt.inputName != "" {
				request.Params.Arguments["name"] = tt.inputName
			}

			// Call handler
			handler := server.HandleCreateEnvironmentTag()
			result, err := handler(context.Background(), request)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				if tt.mockError != nil {
					assert.ErrorContains(t, err, tt.mockError.Error())
				}
				if tt.inputName == "" {
					assert.ErrorContains(t, err, "name")
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)
				assert.Contains(t, textContent.Text,
					fmt.Sprintf("ID: %d", tt.mockID))
			}

			// Verify mock expectations
			mockClient.AssertExpectations(t)
		})
	}
}
