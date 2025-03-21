package models

import (
	"github.com/portainer/client-api-go/v2/pkg/models"
)

type Team struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	MemberIDs []int  `json:"members"`
}

func ConvertToTeam(t *models.PortainerTeam, m []*models.PortainerTeamMembership) Team {
	memberIDs := make([]int, 0)
	for _, member := range m {
		if member.TeamID == t.ID {
			memberIDs = append(memberIDs, int(member.UserID))
		}
	}

	return Team{
		ID:        int(t.ID),
		Name:      t.Name,
		MemberIDs: memberIDs,
	}
}
