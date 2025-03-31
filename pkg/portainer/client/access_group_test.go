package client

import (
	"errors"
	"testing"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	sdkmodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAccessGroups(t *testing.T) {
	tests := []struct {
		name                  string
		mockEndpointGroups    []*sdkmodels.PortainerEndpointGroup
		mockEndpoints         []*sdkmodels.PortainereeEndpoint
		mockEndpointGroupsErr error
		mockEndpointsErr      error
		expected              []models.AccessGroup
		expectedError         bool
	}{
		{
			name: "successful retrieval",
			mockEndpointGroups: []*sdkmodels.PortainerEndpointGroup{
				{
					ID:   1,
					Name: "group1",
					UserAccessPolicies: sdkmodels.PortainerUserAccessPolicies{
						"1": sdkmodels.PortainerAccessPolicy{RoleID: 1}, // environment_administrator
						"2": sdkmodels.PortainerAccessPolicy{RoleID: 2}, // helpdesk_user
						"3": sdkmodels.PortainerAccessPolicy{RoleID: 3}, // standard_user
						"4": sdkmodels.PortainerAccessPolicy{RoleID: 4}, // readonly_user
						"5": sdkmodels.PortainerAccessPolicy{RoleID: 5}, // operator_user
					},
					TeamAccessPolicies: sdkmodels.PortainerTeamAccessPolicies{
						"6":  sdkmodels.PortainerAccessPolicy{RoleID: 1}, // environment_administrator
						"7":  sdkmodels.PortainerAccessPolicy{RoleID: 2}, // helpdesk_user
						"8":  sdkmodels.PortainerAccessPolicy{RoleID: 3}, // standard_user
						"9":  sdkmodels.PortainerAccessPolicy{RoleID: 4}, // readonly_user
						"10": sdkmodels.PortainerAccessPolicy{RoleID: 5}, // operator_user
					},
				},
			},
			mockEndpoints: []*sdkmodels.PortainereeEndpoint{
				{ID: 1, Name: "endpoint1", GroupID: 1},
				{ID: 2, Name: "endpoint2", GroupID: 1},
				{ID: 3, Name: "endpoint3", GroupID: 2},
			},
			expected: []models.AccessGroup{
				{
					ID:             1,
					Name:           "group1",
					EnvironmentIds: []int{1, 2},
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
			},
		},
		{
			name:                  "endpoint group list error",
			mockEndpointGroupsErr: errors.New("failed to list groups"),
			expectedError:         true,
		},
		{
			name: "endpoint list error",
			mockEndpointGroups: []*sdkmodels.PortainerEndpointGroup{
				{ID: 1, Name: "group1"},
			},
			mockEndpointsErr: errors.New("failed to list endpoints"),
			expectedError:    true,
		},
		{
			name:               "empty groups with endpoints",
			mockEndpointGroups: []*sdkmodels.PortainerEndpointGroup{},
			mockEndpoints: []*sdkmodels.PortainereeEndpoint{
				{ID: 1, Name: "endpoint1", GroupID: 1},
				{ID: 2, Name: "endpoint2", GroupID: 2},
			},
			expected: []models.AccessGroup{},
		},
		{
			name: "groups with empty endpoints",
			mockEndpointGroups: []*sdkmodels.PortainerEndpointGroup{
				{
					ID:   1,
					Name: "group1",
					UserAccessPolicies: sdkmodels.PortainerUserAccessPolicies{
						"1": sdkmodels.PortainerAccessPolicy{RoleID: 1},
					},
				},
			},
			mockEndpoints: []*sdkmodels.PortainereeEndpoint{},
			expected: []models.AccessGroup{
				{
					ID:             1,
					Name:           "group1",
					EnvironmentIds: []int{},
					UserAccesses: map[int]string{
						1: "environment_administrator",
					},
					TeamAccesses: map[int]string{},
				},
			},
		},
		{
			name:               "both empty",
			mockEndpointGroups: []*sdkmodels.PortainerEndpointGroup{},
			mockEndpoints:      []*sdkmodels.PortainereeEndpoint{},
			expected:           []models.AccessGroup{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("ListEndpointGroups").Return(tt.mockEndpointGroups, tt.mockEndpointGroupsErr)
			mockAPI.On("ListEndpoints").Return(tt.mockEndpoints, tt.mockEndpointsErr)

			client := &PortainerClient{cli: mockAPI}

			groups, err := client.GetAccessGroups()

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, groups)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestCreateAccessGroup(t *testing.T) {
	tests := []struct {
		name          string
		groupName     string
		envIDs        []int
		mockReturnID  int64
		mockError     error
		expected      int
		expectedError bool
	}{
		{
			name:         "successful creation",
			groupName:    "newgroup",
			envIDs:       []int{1, 2, 3},
			mockReturnID: 1,
			expected:     1,
		},
		{
			name:          "creation error",
			groupName:     "newgroup",
			envIDs:        []int{1},
			mockError:     errors.New("failed to create group"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("CreateEndpointGroup", tt.groupName, mock.Anything).Return(tt.mockReturnID, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			id, err := client.CreateAccessGroup(tt.groupName, tt.envIDs)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, id)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateAccessGroupName(t *testing.T) {
	tests := []struct {
		name          string
		groupID       int
		newName       string
		mockError     error
		expectedError bool
	}{
		{
			name:    "successful update",
			groupID: 1,
			newName: "updated-group",
		},
		{
			name:          "update error",
			groupID:       1,
			newName:       "updated-group",
			mockError:     errors.New("failed to update group"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateEndpointGroup", int64(tt.groupID), &tt.newName, mock.Anything, mock.Anything).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateAccessGroupName(tt.groupID, tt.newName)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateAccessGroupUserAccesses(t *testing.T) {
	tests := []struct {
		name          string
		groupID       int
		userAccesses  map[int]string
		mockError     error
		expectedError bool
	}{
		{
			name:    "successful update",
			groupID: 1,
			userAccesses: map[int]string{
				1: "environment_administrator",
				2: "readonly_user",
			},
		},
		{
			name:    "update error",
			groupID: 1,
			userAccesses: map[int]string{
				1: "environment_administrator",
			},
			mockError:     errors.New("failed to update user accesses"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateEndpointGroup", int64(tt.groupID), mock.Anything, mock.Anything, mock.Anything).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateAccessGroupUserAccesses(tt.groupID, tt.userAccesses)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateAccessGroupTeamAccesses(t *testing.T) {
	tests := []struct {
		name          string
		groupID       int
		teamAccesses  map[int]string
		mockError     error
		expectedError bool
	}{
		{
			name:    "successful update",
			groupID: 1,
			teamAccesses: map[int]string{
				1: "environment_administrator",
				2: "readonly_user",
			},
		},
		{
			name:    "update error",
			groupID: 1,
			teamAccesses: map[int]string{
				1: "environment_administrator",
			},
			mockError:     errors.New("failed to update team accesses"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateEndpointGroup", int64(tt.groupID), mock.Anything, mock.Anything, mock.Anything).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateAccessGroupTeamAccesses(tt.groupID, tt.teamAccesses)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestAddEnvironmentToAccessGroup(t *testing.T) {
	tests := []struct {
		name          string
		groupID       int
		envID         int
		mockError     error
		expectedError bool
	}{
		{
			name:    "successful addition",
			groupID: 1,
			envID:   2,
		},
		{
			name:          "addition error",
			groupID:       1,
			envID:         2,
			mockError:     errors.New("failed to add environment"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("AddEnvironmentToEndpointGroup", int64(tt.groupID), int64(tt.envID)).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.AddEnvironmentToAccessGroup(tt.groupID, tt.envID)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestRemoveEnvironmentFromAccessGroup(t *testing.T) {
	tests := []struct {
		name          string
		groupID       int
		envID         int
		mockError     error
		expectedError bool
	}{
		{
			name:    "successful removal",
			groupID: 1,
			envID:   2,
		},
		{
			name:          "removal error",
			groupID:       1,
			envID:         2,
			mockError:     errors.New("failed to remove environment"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("RemoveEnvironmentFromEndpointGroup", int64(tt.groupID), int64(tt.envID)).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.RemoveEnvironmentFromAccessGroup(tt.groupID, tt.envID)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}
