package client

import (
	"errors"
	"testing"

	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/stretchr/testify/assert"
)

func TestGetUsers(t *testing.T) {
	tests := []struct {
		name          string
		mockUsers     []*apimodels.PortainereeUser
		mockError     error
		expected      []models.User
		expectedError bool
	}{
		{
			name: "successful retrieval - all role types",
			mockUsers: []*apimodels.PortainereeUser{
				{
					ID:       1,
					Username: "admin_user",
					Role:     1, // admin
				},
				{
					ID:       2,
					Username: "regular_user",
					Role:     2, // user
				},
				{
					ID:       3,
					Username: "edge_admin_user",
					Role:     3, // edge_admin
				},
				{
					ID:       4,
					Username: "unknown_role_user",
					Role:     0, // unknown
				},
			},
			expected: []models.User{
				{
					ID:       1,
					Username: "admin_user",
					Role:     models.UserRoleAdmin,
				},
				{
					ID:       2,
					Username: "regular_user",
					Role:     models.UserRoleUser,
				},
				{
					ID:       3,
					Username: "edge_admin_user",
					Role:     models.UserRoleEdgeAdmin,
				},
				{
					ID:       4,
					Username: "unknown_role_user",
					Role:     models.UserRoleUnknown,
				},
			},
		},
		{
			name:      "empty users",
			mockUsers: []*apimodels.PortainereeUser{},
			expected:  []models.User{},
		},
		{
			name:          "list error",
			mockError:     errors.New("failed to list users"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("ListUsers").Return(tt.mockUsers, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			users, err := client.GetUsers()

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, users)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateUserRole(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		role          string
		expectedRole  int64
		mockError     error
		expectedError bool
	}{
		{
			name:         "update to admin role",
			userID:       1,
			role:         models.UserRoleAdmin,
			expectedRole: 1,
		},
		{
			name:         "update to regular user role",
			userID:       2,
			role:         models.UserRoleUser,
			expectedRole: 2,
		},
		{
			name:         "update to edge admin role",
			userID:       3,
			role:         models.UserRoleEdgeAdmin,
			expectedRole: 3,
		},
		{
			name:          "invalid role",
			userID:        4,
			role:          "invalid_role",
			expectedError: true,
		},
		{
			name:          "update error",
			userID:        5,
			role:          models.UserRoleAdmin,
			expectedRole:  1,
			mockError:     errors.New("failed to update user role"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			if !tt.expectedError || tt.mockError != nil {
				mockAPI.On("UpdateUserRole", tt.userID, tt.expectedRole).Return(tt.mockError)
			}

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateUserRole(tt.userID, tt.role)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}
