package client

import (
	"errors"
	"testing"

	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetEnvironments(t *testing.T) {
	tests := []struct {
		name          string
		mockEndpoints []*apimodels.PortainereeEndpoint
		mockError     error
		expected      []models.Environment
		expectedError bool
	}{
		{
			name: "successful retrieval",
			mockEndpoints: []*apimodels.PortainereeEndpoint{
				{
					ID:      1,
					Name:    "env1",
					GroupID: 1,
					Status:  1, // active
					Type:    1, // docker-local
					TagIds:  []int64{1, 2},
					UserAccessPolicies: apimodels.PortainerUserAccessPolicies{
						"1": apimodels.PortainerAccessPolicy{RoleID: 1}, // environment_administrator
						"2": apimodels.PortainerAccessPolicy{RoleID: 2}, // helpdesk_user
						"3": apimodels.PortainerAccessPolicy{RoleID: 3}, // standard_user
						"4": apimodels.PortainerAccessPolicy{RoleID: 4}, // readonly_user
						"5": apimodels.PortainerAccessPolicy{RoleID: 5}, // operator_user
					},
					TeamAccessPolicies: apimodels.PortainerTeamAccessPolicies{
						"6":  apimodels.PortainerAccessPolicy{RoleID: 1}, // environment_administrator
						"7":  apimodels.PortainerAccessPolicy{RoleID: 2}, // helpdesk_user
						"8":  apimodels.PortainerAccessPolicy{RoleID: 3}, // standard_user
						"9":  apimodels.PortainerAccessPolicy{RoleID: 4}, // readonly_user
						"10": apimodels.PortainerAccessPolicy{RoleID: 5}, // operator_user
					},
				},
				{
					ID:      2,
					Name:    "env2",
					GroupID: 1,
					Status:  2, // inactive
					Type:    2, // docker-agent
					TagIds:  []int64{3},
				},
				{
					ID:     3,
					Name:   "env3",
					Status: 0, // unknown
					Type:   0, // unknown
				},
			},
			expected: []models.Environment{
				{
					ID:     1,
					Name:   "env1",
					Status: "active",
					Type:   "docker-local",
					TagIds: []int{1, 2},
					UserAccesses: map[int]string{
						1: "environment_administrator",
						2: "helpdesk_user",
						3: "standard_user",
						4: "readonly_user",
						5: "operator_user",
					},
					TeamAccesses: map[int]string{
						6:  "environment_administrator",
						7:  "helpdesk_user",
						8:  "standard_user",
						9:  "readonly_user",
						10: "operator_user",
					},
				},
				{
					ID:           2,
					Name:         "env2",
					Status:       "inactive",
					Type:         "docker-agent",
					TagIds:       []int{3},
					UserAccesses: map[int]string{},
					TeamAccesses: map[int]string{},
				},
				{
					ID:           3,
					Name:         "env3",
					Status:       "unknown",
					Type:         "unknown",
					TagIds:       []int{},
					UserAccesses: map[int]string{},
					TeamAccesses: map[int]string{},
				},
			},
		},
		{
			name:          "empty environments",
			mockEndpoints: []*apimodels.PortainereeEndpoint{},
			expected:      []models.Environment{},
		},
		{
			name:          "list error",
			mockError:     errors.New("failed to list endpoints"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("ListEndpoints").Return(tt.mockEndpoints, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			environments, err := client.GetEnvironments()

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, environments)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateEnvironmentTags(t *testing.T) {
	tests := []struct {
		name          string
		envID         int
		tagIds        []int
		mockError     error
		expectedError bool
	}{
		{
			name:   "successful update",
			envID:  1,
			tagIds: []int{1, 2, 3},
		},
		{
			name:          "update error",
			envID:         1,
			tagIds:        []int{1},
			mockError:     errors.New("failed to update tags"),
			expectedError: true,
		},
		{
			name:   "empty tags",
			envID:  1,
			tagIds: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateEndpoint", int64(tt.envID), mock.Anything, mock.Anything, mock.Anything).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateEnvironmentTags(tt.envID, tt.tagIds)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateEnvironmentUserAccesses(t *testing.T) {
	tests := []struct {
		name          string
		envID         int
		userAccesses  map[int]string
		mockError     error
		expectedError bool
	}{
		{
			name:  "successful update",
			envID: 1,
			userAccesses: map[int]string{
				1: "environment_administrator",
				2: "helpdesk_user",
				3: "standard_user",
				4: "readonly_user",
				5: "operator_user",
			},
		},
		{
			name:  "update error",
			envID: 1,
			userAccesses: map[int]string{
				1: "environment_administrator",
			},
			mockError:     errors.New("failed to update user accesses"),
			expectedError: true,
		},
		{
			name:         "empty accesses",
			envID:        1,
			userAccesses: map[int]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateEndpoint", int64(tt.envID), mock.Anything, mock.Anything, mock.Anything).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateEnvironmentUserAccesses(tt.envID, tt.userAccesses)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateEnvironmentTeamAccesses(t *testing.T) {
	tests := []struct {
		name          string
		envID         int
		teamAccesses  map[int]string
		mockError     error
		expectedError bool
	}{
		{
			name:  "successful update",
			envID: 1,
			teamAccesses: map[int]string{
				1: "environment_administrator",
				2: "helpdesk_user",
				3: "standard_user",
				4: "readonly_user",
				5: "operator_user",
			},
		},
		{
			name:  "update error",
			envID: 1,
			teamAccesses: map[int]string{
				1: "environment_administrator",
			},
			mockError:     errors.New("failed to update team accesses"),
			expectedError: true,
		},
		{
			name:         "empty accesses",
			envID:        1,
			teamAccesses: map[int]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateEndpoint", int64(tt.envID), mock.Anything, mock.Anything, mock.Anything).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateEnvironmentTeamAccesses(tt.envID, tt.teamAccesses)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}
