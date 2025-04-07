package models

import (
	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
)

type Team struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	MemberIDs []int  `json:"members"`
}

func ConvertToTeam(rawTeam *apimodels.PortainerTeam, rawMemberships []*apimodels.PortainerTeamMembership) Team {
	memberIDs := make([]int, 0)
	for _, member := range rawMemberships {
		if member.TeamID == rawTeam.ID {
			memberIDs = append(memberIDs, int(member.UserID))
		}
	}

	return Team{
		ID:        int(rawTeam.ID),
		Name:      rawTeam.Name,
		MemberIDs: memberIDs,
	}
}
