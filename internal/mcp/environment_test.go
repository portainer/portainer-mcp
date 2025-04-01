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

func TestHandleGetEnvironments(t *testing.T) {
	tests := []struct {
		name             string
		mockEnvironments []models.Environment
		mockError        error
		expectError      bool
	}{
		{
			name: "successful environments retrieval",
			mockEnvironments: []models.Environment{
				{ID: 1, Name: "env1"},
				{ID: 2, Name: "env2"},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:             "api error",
			mockEnvironments: nil,
			mockError:        fmt.Errorf("api error"),
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			mockClient.On("GetEnvironments").Return(tt.mockEnvironments, tt.mockError)

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			handler := server.handleGetEnvironments()
			result, err := handler(context.Background(), mcp.CallToolRequest{})

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

				var environments []models.Environment
				err = json.Unmarshal([]byte(textContent.Text), &environments)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockEnvironments, environments)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleUpdateEnvironmentTags(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		inputTagIDs []int
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:        "successful tags update",
			inputID:     1,
			inputTagIDs: []int{1, 2, 3},
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["tagIds"] = []any{float64(1), float64(2), float64(3)}
			},
		},
		{
			name:        "api error",
			inputID:     1,
			inputTagIDs: []int{1, 2, 3},
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["tagIds"] = []any{float64(1), float64(2), float64(3)}
			},
		},
		{
			name:        "missing id parameter",
			inputID:     0,
			inputTagIDs: []int{1, 2, 3},
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["tagIds"] = []any{float64(1), float64(2), float64(3)}
			},
		},
		{
			name:        "missing tagIds parameter",
			inputID:     1,
			inputTagIDs: nil,
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("UpdateEnvironmentTags", tt.inputID, tt.inputTagIDs).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Arguments: map[string]any{},
				},
			}

			tt.setupParams(&request)

			handler := server.handleUpdateEnvironmentTags()
			result, err := handler(context.Background(), request)

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
				assert.Contains(t, textContent.Text, "successfully")
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleUpdateEnvironmentUserAccesses(t *testing.T) {
	tests := []struct {
		name          string
		inputID       int
		inputAccesses map[int]string
		mockError     error
		expectError   bool
		setupParams   func(request *mcp.CallToolRequest)
	}{
		{
			name:    "successful user accesses update",
			inputID: 1,
			inputAccesses: map[int]string{
				1: "environment_administrator",
				2: "standard_user",
			},
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["userAccesses"] = []any{
					map[string]any{"id": float64(1), "access": "environment_administrator"},
					map[string]any{"id": float64(2), "access": "standard_user"},
				}
			},
		},
		{
			name:    "api error",
			inputID: 1,
			inputAccesses: map[int]string{
				1: "environment_administrator",
			},
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["userAccesses"] = []any{
					map[string]any{"id": float64(1), "access": "environment_administrator"},
				}
			},
		},
		{
			name:        "missing id parameter",
			inputID:     0,
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["userAccesses"] = []any{
					map[string]any{"id": float64(1), "access": "environment_administrator"},
				}
			},
		},
		{
			name:        "missing userAccesses parameter",
			inputID:     1,
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
			},
		},
		{
			name:    "invalid access level",
			inputID: 1,
			inputAccesses: map[int]string{
				1: "invalid_access",
			},
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["userAccesses"] = []any{
					map[string]any{"id": float64(1), "access": "invalid_access"},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("UpdateEnvironmentUserAccesses", tt.inputID, tt.inputAccesses).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Arguments: map[string]any{},
				},
			}

			tt.setupParams(&request)

			handler := server.handleUpdateEnvironmentUserAccesses()
			result, err := handler(context.Background(), request)

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
				assert.Contains(t, textContent.Text, "successfully")
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleUpdateEnvironmentTeamAccesses(t *testing.T) {
	tests := []struct {
		name          string
		inputID       int
		inputAccesses map[int]string
		mockError     error
		expectError   bool
		setupParams   func(request *mcp.CallToolRequest)
	}{
		{
			name:    "successful team accesses update",
			inputID: 1,
			inputAccesses: map[int]string{
				1: "environment_administrator",
				2: "standard_user",
			},
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["teamAccesses"] = []any{
					map[string]any{"id": float64(1), "access": "environment_administrator"},
					map[string]any{"id": float64(2), "access": "standard_user"},
				}
			},
		},
		{
			name:    "api error",
			inputID: 1,
			inputAccesses: map[int]string{
				1: "environment_administrator",
			},
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["teamAccesses"] = []any{
					map[string]any{"id": float64(1), "access": "environment_administrator"},
				}
			},
		},
		{
			name:        "missing id parameter",
			inputID:     0,
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["teamAccesses"] = []any{
					map[string]any{"id": float64(1), "access": "environment_administrator"},
				}
			},
		},
		{
			name:        "missing teamAccesses parameter",
			inputID:     1,
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
			},
		},
		{
			name:    "invalid access level",
			inputID: 1,
			inputAccesses: map[int]string{
				1: "invalid_access",
			},
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["teamAccesses"] = []any{
					map[string]any{"id": float64(1), "access": "invalid_access"},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("UpdateEnvironmentTeamAccesses", tt.inputID, tt.inputAccesses).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := mcp.CallToolRequest{
				Params: struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Arguments: map[string]any{},
				},
			}

			tt.setupParams(&request)

			handler := server.handleUpdateEnvironmentTeamAccesses()
			result, err := handler(context.Background(), request)

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
				assert.Contains(t, textContent.Text, "successfully")
			}

			mockClient.AssertExpectations(t)
		})
	}
}
