package client

import (
	"errors"
	"testing"

	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/stretchr/testify/assert"
)

func TestGetTeams(t *testing.T) {
	tests := []struct {
		name            string
		mockTeams       []*apimodels.PortainerTeam
		mockMemberships []*apimodels.PortainerTeamMembership
		mockTeamError   error
		mockMemberError error
		expected        []models.Team
		expectedError   bool
	}{
		{
			name: "successful retrieval",
			mockTeams: []*apimodels.PortainerTeam{
				{
					ID:   1,
					Name: "team1",
				},
				{
					ID:   2,
					Name: "team2",
				},
			},
			mockMemberships: []*apimodels.PortainerTeamMembership{
				{
					ID:     1,
					UserID: 100,
					TeamID: 1,
				},
				{
					ID:     2,
					UserID: 101,
					TeamID: 1,
				},
				{
					ID:     3,
					UserID: 102,
					TeamID: 2,
				},
			},
			expected: []models.Team{
				{
					ID:        1,
					Name:      "team1",
					MemberIDs: []int{100, 101},
				},
				{
					ID:        2,
					Name:      "team2",
					MemberIDs: []int{102},
				},
			},
		},
		{
			name: "teams with empty memberships",
			mockTeams: []*apimodels.PortainerTeam{
				{
					ID:   1,
					Name: "team1",
				},
				{
					ID:   2,
					Name: "team2",
				},
			},
			mockMemberships: []*apimodels.PortainerTeamMembership{},
			expected: []models.Team{
				{
					ID:        1,
					Name:      "team1",
					MemberIDs: []int{},
				},
				{
					ID:        2,
					Name:      "team2",
					MemberIDs: []int{},
				},
			},
		},
		{
			name:            "empty teams",
			mockTeams:       []*apimodels.PortainerTeam{},
			mockMemberships: []*apimodels.PortainerTeamMembership{},
			expected:        []models.Team{},
		},
		{
			name:          "list teams error",
			mockTeamError: errors.New("failed to list teams"),
			expectedError: true,
		},
		{
			name: "list memberships error",
			mockTeams: []*apimodels.PortainerTeam{
				{
					ID:   1,
					Name: "team1",
				},
			},
			mockMemberError: errors.New("failed to list memberships"),
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("ListTeams").Return(tt.mockTeams, tt.mockTeamError)
			mockAPI.On("ListTeamMemberships").Return(tt.mockMemberships, tt.mockMemberError)

			client := &PortainerClient{cli: mockAPI}

			teams, err := client.GetTeams()

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, teams)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestUpdateTeamName(t *testing.T) {
	tests := []struct {
		name          string
		teamID        int
		teamName      string
		mockError     error
		expectedError bool
	}{
		{
			name:     "successful update",
			teamID:   1,
			teamName: "new-team-name",
		},
		{
			name:          "update error",
			teamID:        2,
			teamName:      "new-team-name",
			mockError:     errors.New("failed to update team name"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("UpdateTeamName", tt.teamID, tt.teamName).Return(tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateTeamName(tt.teamID, tt.teamName)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}

func TestCreateTeam(t *testing.T) {
	tests := []struct {
		name          string
		teamName      string
		mockID        int64
		mockError     error
		expected      int
		expectedError bool
	}{
		{
			name:     "successful creation",
			teamName: "new-team",
			mockID:   1,
			expected: 1,
		},
		{
			name:          "create error",
			teamName:      "new-team",
			mockError:     errors.New("failed to create team"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("CreateTeam", tt.teamName).Return(tt.mockID, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			id, err := client.CreateTeam(tt.teamName)

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

func TestUpdateTeamMembers(t *testing.T) {
	tests := []struct {
		name            string
		teamID          int
		userIDs         []int
		mockMemberships []*apimodels.PortainerTeamMembership
		mockListError   error
		mockDeleteError error
		mockCreateError error
		expectedError   bool
	}{
		{
			name:    "successful update - add and remove members",
			teamID:  1,
			userIDs: []int{101, 102}, // Want to keep 101 and add 102
			mockMemberships: []*apimodels.PortainerTeamMembership{
				{
					ID:     1,
					UserID: 100, // Should be removed
					TeamID: 1,
				},
				{
					ID:     2,
					UserID: 101, // Should be kept
					TeamID: 1,
				},
			},
		},
		{
			name:    "successful update - no changes needed",
			teamID:  1,
			userIDs: []int{100, 101},
			mockMemberships: []*apimodels.PortainerTeamMembership{
				{
					ID:     1,
					UserID: 100,
					TeamID: 1,
				},
				{
					ID:     2,
					UserID: 101,
					TeamID: 1,
				},
			},
		},
		{
			name:          "list memberships error",
			teamID:        1,
			userIDs:       []int{100},
			mockListError: errors.New("failed to list memberships"),
			expectedError: true,
		},
		{
			name:    "delete membership error",
			teamID:  1,
			userIDs: []int{101}, // Want to remove 100
			mockMemberships: []*apimodels.PortainerTeamMembership{
				{
					ID:     1,
					UserID: 100,
					TeamID: 1,
				},
			},
			mockDeleteError: errors.New("failed to delete membership"),
			expectedError:   true,
		},
		{
			name:            "create membership error",
			teamID:          1,
			userIDs:         []int{100}, // Want to add 100
			mockMemberships: []*apimodels.PortainerTeamMembership{},
			mockCreateError: errors.New("failed to create membership"),
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("ListTeamMemberships").Return(tt.mockMemberships, tt.mockListError)

			// Set up delete expectations for memberships that should be removed
			for _, membership := range tt.mockMemberships {
				shouldDelete := true
				for _, keepID := range tt.userIDs {
					if int(membership.UserID) == keepID {
						shouldDelete = false
						break
					}
				}
				if shouldDelete {
					mockAPI.On("DeleteTeamMembership", int(membership.ID)).Return(tt.mockDeleteError)
				}
			}

			// Set up create expectations for new members
			for _, userID := range tt.userIDs {
				exists := false
				for _, membership := range tt.mockMemberships {
					if int(membership.UserID) == userID && int(membership.TeamID) == tt.teamID {
						exists = true
						break
					}
				}
				if !exists {
					mockAPI.On("CreateTeamMembership", tt.teamID, userID).Return(tt.mockCreateError)
				}
			}

			client := &PortainerClient{cli: mockAPI}

			err := client.UpdateTeamMembers(tt.teamID, tt.userIDs)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockAPI.AssertExpectations(t)
		})
	}
}
