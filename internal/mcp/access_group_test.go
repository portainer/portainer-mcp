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

func TestHandleGetAccessGroups(t *testing.T) {
	tests := []struct {
		name        string
		mockGroups  []models.AccessGroup
		mockError   error
		expectError bool
	}{
		{
			name: "successful groups retrieval",
			mockGroups: []models.AccessGroup{
				{ID: 1, Name: "group1"},
				{ID: 2, Name: "group2"},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "api error",
			mockGroups:  nil,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			mockClient.On("GetAccessGroups").Return(tt.mockGroups, tt.mockError)

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			handler := server.handleGetAccessGroups()
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

				var groups []models.AccessGroup
				err = json.Unmarshal([]byte(textContent.Text), &groups)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockGroups, groups)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleCreateAccessGroup(t *testing.T) {
	tests := []struct {
		name        string
		inputName   string
		inputEnvIDs []int
		mockID      int
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:        "successful group creation",
			inputName:   "group1",
			inputEnvIDs: []int{1, 2, 3},
			mockID:      1,
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["environmentIds"] = []any{float64(1), float64(2), float64(3)}
			},
		},
		{
			name:        "api error",
			inputName:   "group1",
			inputEnvIDs: []int{1, 2, 3},
			mockID:      0,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["environmentIds"] = []any{float64(1), float64(2), float64(3)}
			},
		},
		{
			name:        "missing name parameter",
			inputEnvIDs: []int{1, 2, 3},
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["environmentIds"] = []any{float64(1), float64(2), float64(3)}
			},
		},
		{
			name:        "invalid environmentIds - not an array",
			inputName:   "group1",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["environmentIds"] = "not an array"
			},
		},
		{
			name:        "invalid environmentIds - array with non-numbers",
			inputName:   "group1",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["environmentIds"] = []any{"1", "2", "3"}
			},
		},
		{
			name:        "invalid environmentIds - array with mixed types",
			inputName:   "group1",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["environmentIds"] = []any{float64(1), "2", float64(3)}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("CreateAccessGroup", tt.inputName, tt.inputEnvIDs).Return(tt.mockID, tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.handleCreateAccessGroup()
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
				assert.Contains(t, textContent.Text, fmt.Sprintf("ID: %d", tt.mockID))
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleUpdateAccessGroupName(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		inputName   string
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:        "successful name update",
			inputID:     1,
			inputName:   "newname",
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "newname"
			},
		},
		{
			name:        "api error",
			inputID:     1,
			inputName:   "newname",
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "newname"
			},
		},
		{
			name:        "missing id parameter",
			inputName:   "newname",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "newname"
			},
		},
		{
			name:        "missing name parameter",
			inputID:     1,
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
				mockClient.On("UpdateAccessGroupName", tt.inputID, tt.inputName).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.handleUpdateAccessGroupName()
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

func TestHandleUpdateAccessGroupUserAccesses(t *testing.T) {
	tests := []struct {
		name          string
		inputID       int
		inputAccesses []map[string]any
		mockError     error
		expectError   bool
		setupParams   func(request *mcp.CallToolRequest)
	}{
		{
			name:    "successful user accesses update",
			inputID: 1,
			inputAccesses: []map[string]any{
				{"id": float64(1), "access": "environment_administrator"},
				{"id": float64(2), "access": "standard_user"},
			},
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
					"userAccesses": []any{
						map[string]any{"id": float64(1), "access": "environment_administrator"},
						map[string]any{"id": float64(2), "access": "standard_user"},
					},
				}
			},
		},
		{
			name:    "api error",
			inputID: 1,
			inputAccesses: []map[string]any{
				{"id": float64(1), "access": "environment_administrator"},
			},
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
					"userAccesses": []any{
						map[string]any{"id": float64(1), "access": "environment_administrator"},
					},
				}
			},
		},
		{
			name:        "missing id parameter",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"userAccesses": []any{
						map[string]any{"id": float64(1), "access": "environment_administrator"},
					},
				}
			},
		},
		{
			name:        "missing userAccesses parameter",
			inputID:     1,
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
				}
			},
		},
		{
			name:    "invalid access level",
			inputID: 1,
			inputAccesses: []map[string]any{
				{"id": float64(1), "access": "invalid_access"},
			},
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
					"userAccesses": []any{
						map[string]any{"id": float64(1), "access": "invalid_access"},
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				expectedMap := make(map[int]string)
				for _, access := range tt.inputAccesses {
					id := int(access["id"].(float64))
					expectedMap[id] = access["access"].(string)
				}
				mockClient.On("UpdateAccessGroupUserAccesses", tt.inputID, expectedMap).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.handleUpdateAccessGroupUserAccesses()
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

func TestHandleUpdateAccessGroupTeamAccesses(t *testing.T) {
	tests := []struct {
		name          string
		inputID       int
		inputAccesses []map[string]any
		mockError     error
		expectError   bool
		setupParams   func(request *mcp.CallToolRequest)
	}{
		{
			name:    "successful team accesses update",
			inputID: 1,
			inputAccesses: []map[string]any{
				{"id": float64(1), "access": "environment_administrator"},
				{"id": float64(2), "access": "standard_user"},
			},
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
					"teamAccesses": []any{
						map[string]any{"id": float64(1), "access": "environment_administrator"},
						map[string]any{"id": float64(2), "access": "standard_user"},
					},
				}
			},
		},
		{
			name:    "api error",
			inputID: 1,
			inputAccesses: []map[string]any{
				{"id": float64(1), "access": "environment_administrator"},
			},
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
					"teamAccesses": []any{
						map[string]any{"id": float64(1), "access": "environment_administrator"},
					},
				}
			},
		},
		{
			name:        "missing id parameter",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"teamAccesses": []any{
						map[string]any{"id": float64(1), "access": "environment_administrator"},
					},
				}
			},
		},
		{
			name:        "missing teamAccesses parameter",
			inputID:     1,
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
				}
			},
		},
		{
			name:    "invalid access level",
			inputID: 1,
			inputAccesses: []map[string]any{
				{"id": float64(1), "access": "invalid_access"},
			},
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
					"teamAccesses": []any{
						map[string]any{"id": float64(1), "access": "invalid_access"},
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				expectedMap := make(map[int]string)
				for _, access := range tt.inputAccesses {
					id := int(access["id"].(float64))
					expectedMap[id] = access["access"].(string)
				}
				mockClient.On("UpdateAccessGroupTeamAccesses", tt.inputID, expectedMap).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.handleUpdateAccessGroupTeamAccesses()
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

func TestHandleAddEnvironmentToAccessGroup(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		inputEnvID  int
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:        "successful environment addition",
			inputID:     1,
			inputEnvID:  2,
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["environmentId"] = float64(2)
			},
		},
		{
			name:        "api error",
			inputID:     1,
			inputEnvID:  2,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["environmentId"] = float64(2)
			},
		},
		{
			name:        "missing id parameter",
			inputEnvID:  2,
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["environmentId"] = float64(2)
			},
		},
		{
			name:        "missing environmentId parameter",
			inputID:     1,
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
				mockClient.On("AddEnvironmentToAccessGroup", tt.inputID, tt.inputEnvID).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.handleAddEnvironmentToAccessGroup()
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

func TestHandleRemoveEnvironmentFromAccessGroup(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		inputEnvID  int
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:        "successful environment removal",
			inputID:     1,
			inputEnvID:  2,
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["environmentId"] = float64(2)
			},
		},
		{
			name:        "api error",
			inputID:     1,
			inputEnvID:  2,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["environmentId"] = float64(2)
			},
		},
		{
			name:        "missing id parameter",
			inputEnvID:  2,
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["environmentId"] = float64(2)
			},
		},
		{
			name:        "missing environmentId parameter",
			inputID:     1,
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
				mockClient.On("RemoveEnvironmentFromAccessGroup", tt.inputID, tt.inputEnvID).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.handleRemoveEnvironmentFromAccessGroup()
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
