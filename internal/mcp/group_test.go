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

func TestHandleGetEnvironmentGroups(t *testing.T) {
	tests := []struct {
		name        string
		mockGroups  []models.Group
		mockError   error
		expectError bool
	}{
		{
			name: "successful groups retrieval",
			mockGroups: []models.Group{
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
			mockClient.On("GetEnvironmentGroups").Return(tt.mockGroups, tt.mockError)

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			handler := server.HandleGetEnvironmentGroups()
			result, err := handler(context.Background(), mcp.CallToolRequest{})

			if tt.expectError {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError, "result.IsError should be true for API errors")
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok, "Result content should be mcp.TextContent")
				if tt.mockError != nil {
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(mcp.TextContent)
				assert.True(t, ok)

				var groups []models.Group
				err = json.Unmarshal([]byte(textContent.Text), &groups)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockGroups, groups)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleCreateEnvironmentGroup(t *testing.T) {
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
			name:        "missing environmentIds parameter",
			inputName:   "group1",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "group1"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("CreateEnvironmentGroup", tt.inputName, tt.inputEnvIDs).Return(tt.mockID, tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleCreateEnvironmentGroup()
			result, err := handler(context.Background(), request)

			if tt.expectError {
				if tt.mockError != nil {
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.True(t, result.IsError, "result.IsError should be true for API errors")
					assert.Len(t, result.Content, 1)
					textContent, ok := result.Content[0].(mcp.TextContent)
					assert.True(t, ok, "Result content should be mcp.TextContent for API error")
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				} else {
					assert.Error(t, err)
					assert.Nil(t, result)
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

func TestHandleUpdateEnvironmentGroupName(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		inputName   string
		mockError   error
		expectError bool
		setupParams func(request mcp.CallToolRequest) mcp.CallToolRequest
	}{
		{
			name:        "successful name update",
			inputID:     1,
			inputName:   "newname",
			mockError:   nil,
			expectError: false,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "newname"
				return request
			},
		},
		{
			name:        "api error",
			inputID:     1,
			inputName:   "newname",
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "newname"
				return request
			},
		},
		{
			name:        "missing id parameter",
			inputName:   "newname",
			mockError:   nil,
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["name"] = "newname"
				return request
			},
		},
		{
			name:        "missing name parameter",
			inputID:     1,
			mockError:   nil,
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				return request
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("UpdateEnvironmentGroupName", tt.inputID, tt.inputName).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			request = tt.setupParams(request)

			handler := server.HandleUpdateEnvironmentGroupName()
			result, err := handler(context.Background(), request)

			if tt.expectError {
				if tt.mockError != nil {
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.True(t, result.IsError, "result.IsError should be true for API errors")
					assert.Len(t, result.Content, 1)
					textContent, ok := result.Content[0].(mcp.TextContent)
					assert.True(t, ok, "Result content should be mcp.TextContent for API error")
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				} else {
					assert.Error(t, err)
					assert.Nil(t, result)
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

func TestHandleUpdateEnvironmentGroupEnvironments(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		inputName   string
		inputEnvIDs []int
		mockError   error
		expectError bool
		setupParams func(request mcp.CallToolRequest) mcp.CallToolRequest
	}{
		{
			name:        "successful environments update",
			inputID:     1,
			inputName:   "group1",
			inputEnvIDs: []int{1, 2, 3},
			mockError:   nil,
			expectError: false,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["environmentIds"] = []any{float64(1), float64(2), float64(3)}
				return request
			},
		},
		{
			name:        "api error",
			inputID:     1,
			inputName:   "group1",
			inputEnvIDs: []int{1, 2, 3},
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["environmentIds"] = []any{float64(1), float64(2), float64(3)}
				return request
			},
		},
		{
			name:        "missing id parameter",
			inputName:   "group1",
			inputEnvIDs: []int{1, 2, 3},
			mockError:   nil,
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["environmentIds"] = []any{float64(1), float64(2), float64(3)}
				return request
			},
		},
		{
			name:        "missing name parameter",
			inputID:     1,
			inputEnvIDs: []int{1, 2, 3},
			mockError:   nil,
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["environmentIds"] = []any{float64(1), float64(2), float64(3)}
				return request
			},
		},
		{
			name:        "missing environmentIds parameter",
			inputID:     1,
			inputName:   "group1",
			mockError:   nil,
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "group1"
				return request
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("UpdateEnvironmentGroupEnvironments", tt.inputID, tt.inputName, tt.inputEnvIDs).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			request = tt.setupParams(request)

			handler := server.HandleUpdateEnvironmentGroupEnvironments()
			result, err := handler(context.Background(), request)

			if tt.expectError {
				if tt.mockError != nil {
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.True(t, result.IsError, "result.IsError should be true for API errors")
					assert.Len(t, result.Content, 1)
					textContent, ok := result.Content[0].(mcp.TextContent)
					assert.True(t, ok, "Result content should be mcp.TextContent for API error")
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				} else {
					assert.Error(t, err)
					assert.Nil(t, result)
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

func TestHandleUpdateEnvironmentGroupTags(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		inputName   string
		inputTagIDs []int
		mockError   error
		expectError bool
		setupParams func(request mcp.CallToolRequest) mcp.CallToolRequest
	}{
		{
			name:        "successful tags update",
			inputID:     1,
			inputName:   "group1",
			inputTagIDs: []int{1, 2, 3},
			mockError:   nil,
			expectError: false,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["tagIds"] = []any{float64(1), float64(2), float64(3)}
				return request
			},
		},
		{
			name:        "api error",
			inputID:     1,
			inputName:   "group1",
			inputTagIDs: []int{1, 2, 3},
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["tagIds"] = []any{float64(1), float64(2), float64(3)}
				return request
			},
		},
		{
			name:        "missing id parameter",
			inputName:   "group1",
			inputTagIDs: []int{1, 2, 3},
			mockError:   nil,
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["name"] = "group1"
				request.Params.Arguments["tagIds"] = []any{float64(1), float64(2), float64(3)}
				return request
			},
		},
		{
			name:        "missing name parameter",
			inputID:     1,
			inputTagIDs: []int{1, 2, 3},
			mockError:   nil,
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["tagIds"] = []any{float64(1), float64(2), float64(3)}
				return request
			},
		},
		{
			name:        "missing tagIds parameter",
			inputID:     1,
			inputName:   "group1",
			mockError:   nil,
			expectError: true,
			setupParams: func(request mcp.CallToolRequest) mcp.CallToolRequest {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "group1"
				return request
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			if !tt.expectError || tt.mockError != nil {
				mockClient.On("UpdateEnvironmentGroupTags", tt.inputID, tt.inputName, tt.inputTagIDs).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			request = tt.setupParams(request)

			handler := server.HandleUpdateEnvironmentGroupTags()
			result, err := handler(context.Background(), request)

			if tt.expectError {
				if tt.mockError != nil {
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.True(t, result.IsError, "result.IsError should be true for API errors")
					assert.Len(t, result.Content, 1)
					textContent, ok := result.Content[0].(mcp.TextContent)
					assert.True(t, ok, "Result content should be mcp.TextContent for API error")
					assert.Contains(t, textContent.Text, tt.mockError.Error())
				} else {
					assert.Error(t, err)
					assert.Nil(t, result)
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
