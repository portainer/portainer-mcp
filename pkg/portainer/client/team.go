package client

import (
	"fmt"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
)

// GetTeams retrieves all teams from the Portainer server.
//
// Returns:
//   - A slice of Team objects containing team information
//   - An error if the operation fails
func (c *PortainerClient) GetTeams() ([]models.Team, error) {
	portainerTeams, err := c.cli.ListTeams()
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	// Get team memberships to populate team members
	memberships, err := c.cli.ListTeamMemberships()
	if err != nil {
		return nil, fmt.Errorf("failed to list team memberships: %w", err)
	}

	teams := make([]models.Team, len(portainerTeams))
	for i, team := range portainerTeams {
		teams[i] = models.ConvertToTeam(team, memberships)
	}

	return teams, nil
}

// UpdateTeam updates the name of a team.
//
// Parameters:
//   - id: The ID of the team to update
//   - name: The new name for the team
func (c *PortainerClient) UpdateTeam(id int, name string) error {
	return c.cli.UpdateTeam(id, name)
}

// CreateTeam creates a new team.
//
// Parameters:
//   - name: The name of the team
//
// Returns:
//   - The ID of the created team
//   - An error if the operation fails
func (c *PortainerClient) CreateTeam(name string) (int, error) {
	id, err := c.cli.CreateTeam(name)
	if err != nil {
		return 0, fmt.Errorf("failed to create team: %w", err)
	}

	return int(id), nil
}

// UpdateTeamMembers updates the members of a team.
//
// Parameters:
//   - teamId: The ID of the team to update
//   - userIds: The IDs of the users associated with the team
func (c *PortainerClient) UpdateTeamMembers(teamId int, userIds []int) error {
	memberships, err := c.cli.ListTeamMemberships()
	if err != nil {
		return fmt.Errorf("failed to list team memberships: %w", err)
	}

	// Track which users are already members of the team
	existingMembers := make(map[int]bool)

	// First, handle existing memberships
	for _, membership := range memberships {
		if membership.TeamID == int64(teamId) {
			userID := membership.UserID
			existingMembers[int(userID)] = true

			// Check if this user should remain in the team
			shouldKeep := false
			for _, id := range userIds {
				if id == int(userID) {
					shouldKeep = true
					break
				}
			}

			// If user should not remain in the team, delete the membership
			if !shouldKeep {
				if err := c.cli.DeleteTeamMembership(int(membership.ID)); err != nil {
					return fmt.Errorf("failed to delete team membership for user %d: %w", userID, err)
				}
			}
		}
	}

	// Then, create memberships for new users
	for _, userID := range userIds {
		// Skip if user is already a member
		if existingMembers[userID] {
			continue
		}

		// Create new membership for this user
		if err := c.cli.CreateTeamMembership(teamId, userID); err != nil {
			return fmt.Errorf("failed to create team membership for user %d: %w", userID, err)
		}
	}

	return nil
}
