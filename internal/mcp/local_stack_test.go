package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/stretchr/testify/assert"
)

func TestHandleGetLocalStacks(t *testing.T) {
	tests := []struct {
		name        string
		mockStacks  []models.LocalStack
		mockError   error
		expectError bool
	}{
		{
			name: "successful retrieval",
			mockStacks: []models.LocalStack{
				{ID: 1, Name: "stack1", Type: "compose", Status: "active", EndpointID: 3},
				{ID: 2, Name: "stack2", Type: "compose", Status: "inactive", EndpointID: 3},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "empty stacks",
			mockStacks:  []models.LocalStack{},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "api error",
			mockStacks:  nil,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			mockClient.On("GetLocalStacks").Return(tt.mockStacks, tt.mockError)

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			handler := server.HandleGetLocalStacks()
			result, err := handler(context.Background(), mcp.CallToolRequest{})

			if tt.expectError {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError, "result.IsError should be true for expected errors")
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok, "Result content should be mcp.TextContent for errors")
				assert.Contains(t, textContent.Text, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)

				var stacks []models.LocalStack
				err = json.Unmarshal([]byte(textContent.Text), &stacks)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockStacks, stacks)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleGetLocalStackFile(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		mockContent string
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:        "successful file retrieval",
			inputID:     1,
			mockContent: "services:\n  web:\n    image: nginx",
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
				}
			},
		},
		{
			name:        "api error",
			inputID:     1,
			mockContent: "",
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
				}
			},
		},
		{
			name:        "missing id parameter",
			inputID:     0,
			mockContent: "",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("GetLocalStackFile", tt.inputID).Return(tt.mockContent, tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleGetLocalStackFile()
			result, err := handler(context.Background(), request)

			if tt.expectError {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError, "result.IsError should be true for expected errors")
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok, "Result content should be mcp.TextContent for errors")
				if tt.mockError != nil {
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				} else {
					assert.NotEmpty(t, textContent.Text, "Error message should not be empty for parameter errors")
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)
				assert.Equal(t, tt.mockContent, textContent.Text)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleCreateLocalStack(t *testing.T) {
	tests := []struct {
		name        string
		endpointId  int
		inputName   string
		inputFile   string
		inputEnv    []models.LocalStackEnvVar
		mockID      int
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:       "successful creation",
			endpointId: 3,
			inputName:  "test-stack",
			inputFile:  "services:\n  web:\n    image: nginx",
			inputEnv:   []models.LocalStackEnvVar{},
			mockID:     10,
			mockError:  nil,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"environmentId": float64(3),
					"name":          "test-stack",
					"file":          "services:\n  web:\n    image: nginx",
				}
			},
		},
		{
			name:       "successful creation with env vars",
			endpointId: 3,
			inputName:  "test-stack",
			inputFile:  "services:\n  web:\n    image: nginx",
			inputEnv:   []models.LocalStackEnvVar{{Name: "DB_HOST", Value: "localhost"}},
			mockID:     11,
			mockError:  nil,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"environmentId": float64(3),
					"name":          "test-stack",
					"file":          "services:\n  web:\n    image: nginx",
					"env": []any{
						map[string]any{"name": "DB_HOST", "value": "localhost"},
					},
				}
			},
		},
		{
			name:        "api error",
			endpointId:  3,
			inputName:   "test-stack",
			inputFile:   "services:\n  web:\n    image: nginx",
			inputEnv:    []models.LocalStackEnvVar{},
			mockID:      0,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"environmentId": float64(3),
					"name":          "test-stack",
					"file":          "services:\n  web:\n    image: nginx",
				}
			},
		},
		{
			name:        "missing name parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"environmentId": float64(3),
					"file":          "services:\n  web:\n    image: nginx",
				}
			},
		},
		{
			name:        "missing file parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"environmentId": float64(3),
					"name":          "test-stack",
				}
			},
		},
		{
			name:        "missing environmentId parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"name": "test-stack",
					"file": "services:\n  web:\n    image: nginx",
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("CreateLocalStack", tt.endpointId, tt.inputName, tt.inputFile, tt.inputEnv).Return(tt.mockID, tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleCreateLocalStack()
			result, err := handler(context.Background(), request)

			if tt.expectError {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError, "result.IsError should be true for expected errors")
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok, "Result content should be mcp.TextContent for errors")
				if tt.mockError != nil {
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				} else {
					assert.NotEmpty(t, textContent.Text, "Error message should not be empty for parameter errors")
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

func TestHandleUpdateLocalStack(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		endpointId  int
		inputFile   string
		inputEnv    []models.LocalStackEnvVar
		prune       bool
		pullImage   bool
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:       "successful update",
			inputID:    1,
			endpointId: 3,
			inputFile:  "services:\n  web:\n    image: nginx:latest",
			inputEnv:   []models.LocalStackEnvVar{},
			prune:      false,
			pullImage:  false,
			mockError:  nil,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":            float64(1),
					"environmentId": float64(3),
					"file":          "services:\n  web:\n    image: nginx:latest",
				}
			},
		},
		{
			name:       "successful update with options",
			inputID:    1,
			endpointId: 3,
			inputFile:  "services:\n  web:\n    image: nginx:latest",
			inputEnv:   []models.LocalStackEnvVar{{Name: "KEY", Value: "val"}},
			prune:      true,
			pullImage:  true,
			mockError:  nil,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":            float64(1),
					"environmentId": float64(3),
					"file":          "services:\n  web:\n    image: nginx:latest",
					"env":           []any{map[string]any{"name": "KEY", "value": "val"}},
					"prune":         true,
					"pullImage":     true,
				}
			},
		},
		{
			name:        "api error",
			inputID:     1,
			endpointId:  3,
			inputFile:   "services:\n  web:\n    image: nginx",
			inputEnv:    []models.LocalStackEnvVar{},
			prune:       false,
			pullImage:   false,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":            float64(1),
					"environmentId": float64(3),
					"file":          "services:\n  web:\n    image: nginx",
				}
			},
		},
		{
			name:        "missing id parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"environmentId": float64(3),
					"file":          "services:\n  web:\n    image: nginx",
				}
			},
		},
		{
			name:        "missing environmentId parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":   float64(1),
					"file": "services:\n  web:\n    image: nginx",
				}
			},
		},
		{
			name:        "missing file parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":            float64(1),
					"environmentId": float64(3),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("UpdateLocalStack", tt.inputID, tt.endpointId, tt.inputFile, tt.inputEnv, tt.prune, tt.pullImage).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleUpdateLocalStack()
			result, err := handler(context.Background(), request)

			if tt.expectError {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError, "result.IsError should be true for expected errors")
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok, "Result content should be mcp.TextContent for errors")
				if tt.mockError != nil {
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				} else {
					assert.NotEmpty(t, textContent.Text, "Error message should not be empty for parameter errors")
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

func TestHandleStartLocalStack(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		endpointId  int
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:       "successful start",
			inputID:    1,
			endpointId: 3,
			mockError:  nil,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":            float64(1),
					"environmentId": float64(3),
				}
			},
		},
		{
			name:        "api error",
			inputID:     1,
			endpointId:  3,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":            float64(1),
					"environmentId": float64(3),
				}
			},
		},
		{
			name:        "missing id parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"environmentId": float64(3),
				}
			},
		},
		{
			name:        "missing environmentId parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("StartLocalStack", tt.inputID, tt.endpointId).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleStartLocalStack()
			result, err := handler(context.Background(), request)

			if tt.expectError {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError, "result.IsError should be true for expected errors")
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok, "Result content should be mcp.TextContent for errors")
				if tt.mockError != nil {
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				} else {
					assert.NotEmpty(t, textContent.Text)
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

func TestHandleStopLocalStack(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		endpointId  int
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:       "successful stop",
			inputID:    1,
			endpointId: 3,
			mockError:  nil,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":            float64(1),
					"environmentId": float64(3),
				}
			},
		},
		{
			name:        "api error",
			inputID:     1,
			endpointId:  3,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":            float64(1),
					"environmentId": float64(3),
				}
			},
		},
		{
			name:        "missing id parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"environmentId": float64(3),
				}
			},
		},
		{
			name:        "missing environmentId parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("StopLocalStack", tt.inputID, tt.endpointId).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleStopLocalStack()
			result, err := handler(context.Background(), request)

			if tt.expectError {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError, "result.IsError should be true for expected errors")
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok, "Result content should be mcp.TextContent for errors")
				if tt.mockError != nil {
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				} else {
					assert.NotEmpty(t, textContent.Text)
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

func TestHandleDeleteLocalStack(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		endpointId  int
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:       "successful delete",
			inputID:    1,
			endpointId: 3,
			mockError:  nil,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":            float64(1),
					"environmentId": float64(3),
				}
			},
		},
		{
			name:        "api error",
			inputID:     1,
			endpointId:  3,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id":            float64(1),
					"environmentId": float64(3),
				}
			},
		},
		{
			name:        "missing id parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"environmentId": float64(3),
				}
			},
		},
		{
			name:        "missing environmentId parameter",
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments = map[string]any{
					"id": float64(1),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("DeleteLocalStack", tt.inputID, tt.endpointId).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleDeleteLocalStack()
			result, err := handler(context.Background(), request)

			if tt.expectError {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError, "result.IsError should be true for expected errors")
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok, "Result content should be mcp.TextContent for errors")
				if tt.mockError != nil {
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				} else {
					assert.NotEmpty(t, textContent.Text)
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
