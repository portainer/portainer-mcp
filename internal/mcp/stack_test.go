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

func TestHandleGetStacks(t *testing.T) {
	tests := []struct {
		name        string
		mockStacks  []models.Stack
		mockError   error
		expectError bool
	}{
		{
			name: "successful stacks retrieval",
			mockStacks: []models.Stack{
				{ID: 1, Name: "stack1"},
				{ID: 2, Name: "stack2"},
			},
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
			mockClient.On("GetStacks").Return(tt.mockStacks, tt.mockError)

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			handler := server.HandleGetStacks()
			result, err := handler(context.Background(), mcp.CallToolRequest{})

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

				var stacks []models.Stack
				err = json.Unmarshal([]byte(textContent.Text), &stacks)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockStacks, stacks)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleGetStackFile(t *testing.T) {
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
			mockContent: "version: '3'\nservices:\n  web:\n    image: nginx",
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
			},
		},
		{
			name:        "api error",
			inputID:     1,
			mockContent: "",
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
			},
		},
		{
			name:        "missing id parameter",
			inputID:     0,
			mockContent: "",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				// No need to set any parameters as the request will be invalid
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("GetStackFile", tt.inputID).Return(tt.mockContent, tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleGetStackFile()
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

func TestHandleCreateStack(t *testing.T) {
	tests := []struct {
		name             string
		inputName        string
		inputFile        string
		inputEnvGroupIDs []int
		mockID           int
		mockError        error
		expectError      bool
		setupParams      func(request *mcp.CallToolRequest)
	}{
		{
			name:             "successful stack creation",
			inputName:        "test-stack",
			inputFile:        "version: '3'\nservices:\n  web:\n    image: nginx",
			inputEnvGroupIDs: []int{1, 2},
			mockID:           1,
			mockError:        nil,
			expectError:      false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "test-stack"
				request.Params.Arguments["file"] = "version: '3'\nservices:\n  web:\n    image: nginx"
				request.Params.Arguments["environmentGroupIds"] = []any{float64(1), float64(2)}
			},
		},
		{
			name:             "api error",
			inputName:        "test-stack",
			inputFile:        "version: '3'\nservices:\n  web:\n    image: nginx",
			inputEnvGroupIDs: []int{1, 2},
			mockID:           0,
			mockError:        fmt.Errorf("api error"),
			expectError:      true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "test-stack"
				request.Params.Arguments["file"] = "version: '3'\nservices:\n  web:\n    image: nginx"
				request.Params.Arguments["environmentGroupIds"] = []any{float64(1), float64(2)}
			},
		},
		{
			name:             "missing name parameter",
			inputName:        "",
			inputFile:        "version: '3'\nservices:\n  web:\n    image: nginx",
			inputEnvGroupIDs: []int{1, 2},
			mockID:           0,
			mockError:        nil,
			expectError:      true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["file"] = "version: '3'\nservices:\n  web:\n    image: nginx"
				request.Params.Arguments["environmentGroupIds"] = []any{float64(1), float64(2)}
			},
		},
		{
			name:             "missing file parameter",
			inputName:        "test-stack",
			inputFile:        "",
			inputEnvGroupIDs: []int{1, 2},
			mockID:           0,
			mockError:        nil,
			expectError:      true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "test-stack"
				request.Params.Arguments["environmentGroupIds"] = []any{float64(1), float64(2)}
			},
		},
		{
			name:             "missing environmentGroupIds parameter",
			inputName:        "test-stack",
			inputFile:        "version: '3'\nservices:\n  web:\n    image: nginx",
			inputEnvGroupIDs: nil,
			mockID:           0,
			mockError:        nil,
			expectError:      true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "test-stack"
				request.Params.Arguments["file"] = "version: '3'\nservices:\n  web:\n    image: nginx"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("CreateStack", tt.inputName, tt.inputFile, tt.inputEnvGroupIDs).Return(tt.mockID, tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleCreateStack()
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

func TestHandleUpdateStack(t *testing.T) {
	tests := []struct {
		name             string
		inputID          int
		inputFile        string
		inputEnvGroupIDs []int
		mockError        error
		expectError      bool
		setupParams      func(request *mcp.CallToolRequest)
	}{
		{
			name:             "successful stack update",
			inputID:          1,
			inputFile:        "version: '3'\nservices:\n  web:\n    image: nginx",
			inputEnvGroupIDs: []int{1, 2},
			mockError:        nil,
			expectError:      false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["file"] = "version: '3'\nservices:\n  web:\n    image: nginx"
				request.Params.Arguments["environmentGroupIds"] = []any{float64(1), float64(2)}
			},
		},
		{
			name:             "api error",
			inputID:          1,
			inputFile:        "version: '3'\nservices:\n  web:\n    image: nginx",
			inputEnvGroupIDs: []int{1, 2},
			mockError:        fmt.Errorf("api error"),
			expectError:      true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["file"] = "version: '3'\nservices:\n  web:\n    image: nginx"
				request.Params.Arguments["environmentGroupIds"] = []any{float64(1), float64(2)}
			},
		},
		{
			name:             "missing id parameter",
			inputID:          0,
			inputFile:        "version: '3'\nservices:\n  web:\n    image: nginx",
			inputEnvGroupIDs: []int{1, 2},
			mockError:        nil,
			expectError:      true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["file"] = "version: '3'\nservices:\n  web:\n    image: nginx"
				request.Params.Arguments["environmentGroupIds"] = []any{float64(1), float64(2)}
			},
		},
		{
			name:             "missing file parameter",
			inputID:          1,
			inputFile:        "",
			inputEnvGroupIDs: []int{1, 2},
			mockError:        nil,
			expectError:      true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["environmentGroupIds"] = []any{float64(1), float64(2)}
			},
		},
		{
			name:             "missing environmentGroupIds parameter",
			inputID:          1,
			inputFile:        "version: '3'\nservices:\n  web:\n    image: nginx",
			inputEnvGroupIDs: nil,
			mockError:        nil,
			expectError:      true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["file"] = "version: '3'\nservices:\n  web:\n    image: nginx"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("UpdateStack", tt.inputID, tt.inputFile, tt.inputEnvGroupIDs).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleUpdateStack()
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
