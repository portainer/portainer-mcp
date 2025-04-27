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

func TestHandleCreateTeam(t *testing.T) {
	tests := []struct {
		name        string
		teamName    string
		mockID      int
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:        "successful team creation",
			teamName:    "test-team",
			mockID:      1,
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "test-team"
			},
		},
		{
			name:        "api error",
			teamName:    "test-team",
			mockID:      0,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "test-team"
			},
		},
		{
			name:        "missing name parameter",
			teamName:    "",
			mockID:      0,
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
				mockClient.On("CreateTeam", tt.teamName).Return(tt.mockID, tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleCreateTeam()
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

func TestHandleGetTeams(t *testing.T) {
	tests := []struct {
		name        string
		mockTeams   []models.Team
		mockError   error
		expectError bool
	}{
		{
			name: "successful teams retrieval",
			mockTeams: []models.Team{
				{ID: 1, Name: "team1"},
				{ID: 2, Name: "team2"},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "api error",
			mockTeams:   nil,
			mockError:   fmt.Errorf("api error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockPortainerClient{}
			mockClient.On("GetTeams").Return(tt.mockTeams, tt.mockError)

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			handler := server.HandleGetTeams()
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

				var teams []models.Team
				err = json.Unmarshal([]byte(textContent.Text), &teams)
				assert.NoError(t, err)
				assert.Equal(t, tt.mockTeams, teams)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestHandleUpdateTeamName(t *testing.T) {
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
			inputName:   "new-name",
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "new-name"
			},
		},
		{
			name:        "api error",
			inputID:     1,
			inputName:   "new-name",
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["name"] = "new-name"
			},
		},
		{
			name:        "missing id parameter",
			inputID:     0,
			inputName:   "new-name",
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["name"] = "new-name"
			},
		},
		{
			name:        "missing name parameter",
			inputID:     1,
			inputName:   "",
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
				mockClient.On("UpdateTeamName", tt.inputID, tt.inputName).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleUpdateTeamName()
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

func TestHandleUpdateTeamMembers(t *testing.T) {
	tests := []struct {
		name        string
		inputID     int
		inputUsers  []int
		mockError   error
		expectError bool
		setupParams func(request *mcp.CallToolRequest)
	}{
		{
			name:        "successful members update",
			inputID:     1,
			inputUsers:  []int{1, 2, 3},
			mockError:   nil,
			expectError: false,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["userIds"] = []any{float64(1), float64(2), float64(3)}
			},
		},
		{
			name:        "api error",
			inputID:     1,
			inputUsers:  []int{1, 2, 3},
			mockError:   fmt.Errorf("api error"),
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["id"] = float64(1)
				request.Params.Arguments["userIds"] = []any{float64(1), float64(2), float64(3)}
			},
		},
		{
			name:        "missing id parameter",
			inputID:     0,
			inputUsers:  []int{1, 2, 3},
			mockError:   nil,
			expectError: true,
			setupParams: func(request *mcp.CallToolRequest) {
				request.Params.Arguments["userIds"] = []any{float64(1), float64(2), float64(3)}
			},
		},
		{
			name:        "missing userIds parameter",
			inputID:     1,
			inputUsers:  nil,
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
				mockClient.On("UpdateTeamMembers", tt.inputID, tt.inputUsers).Return(tt.mockError)
			}

			server := &PortainerMCPServer{
				cli: mockClient,
			}

			request := CreateMCPRequest(map[string]any{})
			tt.setupParams(&request)

			handler := server.HandleUpdateTeamMembers()
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
