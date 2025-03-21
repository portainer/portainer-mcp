package models

import (
	"reflect"
	"testing"

	"github.com/portainer/client-api-go/v2/pkg/models"
)

func TestConvertToTeam(t *testing.T) {
	tests := []struct {
		name         string
		team         *models.PortainerTeam
		memberships  []*models.PortainerTeamMembership
		expectedTeam Team
	}{
		{
			name: "team with multiple members",
			team: &models.PortainerTeam{
				ID:   1,
				Name: "DevOps",
			},
			memberships: []*models.PortainerTeamMembership{
				{TeamID: 1, UserID: 100},
				{TeamID: 1, UserID: 101},
				{TeamID: 1, UserID: 102},
				{TeamID: 2, UserID: 200}, // Different team, should be ignored
			},
			expectedTeam: Team{
				ID:        1,
				Name:      "DevOps",
				MemberIDs: []int{100, 101, 102},
			},
		},
		{
			name: "team with no members",
			team: &models.PortainerTeam{
				ID:   2,
				Name: "Empty Team",
			},
			memberships: []*models.PortainerTeamMembership{
				{TeamID: 1, UserID: 100}, // Different team
				{TeamID: 3, UserID: 300}, // Different team
			},
			expectedTeam: Team{
				ID:        2,
				Name:      "Empty Team",
				MemberIDs: []int{},
			},
		},
		{
			name: "team with single member",
			team: &models.PortainerTeam{
				ID:   3,
				Name: "Solo Team",
			},
			memberships: []*models.PortainerTeamMembership{
				{TeamID: 3, UserID: 300},
			},
			expectedTeam: Team{
				ID:        3,
				Name:      "Solo Team",
				MemberIDs: []int{300},
			},
		},
		{
			name: "team with empty memberships list",
			team: &models.PortainerTeam{
				ID:   4,
				Name: "New Team",
			},
			memberships: []*models.PortainerTeamMembership{},
			expectedTeam: Team{
				ID:        4,
				Name:      "New Team",
				MemberIDs: []int{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToTeam(tt.team, tt.memberships)

			if result.ID != tt.expectedTeam.ID {
				t.Errorf("ID mismatch: got %v, want %v", result.ID, tt.expectedTeam.ID)
			}

			if result.Name != tt.expectedTeam.Name {
				t.Errorf("Name mismatch: got %v, want %v", result.Name, tt.expectedTeam.Name)
			}

			if !reflect.DeepEqual(result.MemberIDs, tt.expectedTeam.MemberIDs) {
				t.Errorf("MemberIDs mismatch: got %v, want %v", result.MemberIDs, tt.expectedTeam.MemberIDs)
			}
		})
	}
}
