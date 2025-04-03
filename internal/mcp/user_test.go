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

func TestHandleGetUsers(t *testing.T) {
	tests := []struct {
		name        string
		mockUsers   []models.User
		mockError   error
		expectError bool
	}{
		{
			name: "successful users retrieval",
			mockUsers: []models.User{
				{ID: 1, Username: "user1", Role: "admin"},
				{ID: 2, Username: "user2", Role: "user"},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "api error",
			mockUsers:   nil,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockPortainerClient{}
			mockClient.On("GetUsers").Return(tt.mockUsers, tt.mockError)

			// Create server with mock client
			server := &PortainerMCPServer{
				cli: mockClient,
			}

			// Call handler
			handler := server.handleGetUsers()
			result, err := handler(context.Background(), mcp.CallToolRequest{})

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				if tt.mockError != nil {
					assert.ErrorContains(t, err, tt.mockError.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)

				var users []models.User
				err = json.Unmarshal([]byte(textContent.Text), &users)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockUsers, users)
			}

			// Verify mock expectations
			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleUpdateUserRole(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		inputRole   string
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:        "successful role update",
			inputID:     1,
			inputRole:   "admin",
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["role"] = "admin"
			},
		},
		{
			name:        "api error",
			inputID:     1,
			inputRole:   "admin",
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["role"] = "admin"
			},
		},
		{
			name:        "missing id parameter",
			inputID:     0,
			inputRole:   "admin",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["role"] = "admin"
			},
		},
		{
			name:        "missing role parameter",
			inputID:     1,
			inputRole:   "",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
			},
		},
		{
			name:        "invalid role",
			inputID:     1,
			inputRole:   "invalid_role",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["role"] = "invalid_role"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("UpdateUserRole", tt.inputID, tt.inputRole).Return(tt.mockError)
			}

			// Create server with mock client
			server := &PortainerMCPServer{
				cli: mockClient,
			}

			// Create request with parameters
			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			// Call handler
			handler := server.handleUpdateUserRole()
			result, err := handler(context.Background(), request)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				if tt.mockError != nil {
					assert.ErrorContains(t, err, tt.mockError.Error())
				}
				if tt.inputRole == "invalid_role" {
					assert.ErrorContains(t, err, "invalid role")
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)
				assert.Contains(t, textContent.Text, "successfully")
			}

			// Verify mock expectations
			mockClient.AssertExpectations(t)
		})
	}
}
