package models

import (
	"github.com/portainer/client-api-go/v2/pkg/models"
)

type AccessGroup struct {
	ID             int            `json:"id"`
	Name           string         `json:"name"`
	EnvironmentIds []int          `json:"environment_ids"`
	UserAccesses   map[int]string `json:"user_accesses"`
	TeamAccesses   map[int]string `json:"team_accesses"`
}

func ConvertEndpointGroupToAccessGroup(g *models.PortainerEndpointGroup, envs []*models.PortainereeEndpoint) AccessGroup {
	environmentIds := make([]int, 0)
	for _, env := range envs {
		if env.GroupID == g.ID {
			environmentIds = append(environmentIds, int(env.ID))
		}
	}

	return AccessGroup{
		ID:             int(g.ID),
		Name:           g.Name,
		EnvironmentIds: environmentIds,
		UserAccesses:   convertAccesses(g.UserAccessPolicies),
		TeamAccesses:   convertAccesses(g.TeamAccessPolicies),
	}
}
