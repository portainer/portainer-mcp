package models

import (
	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
)

type AccessGroup struct {
	ID             int            `json:"id"`
	Name           string         `json:"name"`
	EnvironmentIds []int          `json:"environment_ids"`
	UserAccesses   map[int]string `json:"user_accesses"`
	TeamAccesses   map[int]string `json:"team_accesses"`
}

func ConvertEndpointGroupToAccessGroup(rawGroup *apimodels.PortainerEndpointGroup, rawEndpoints []*apimodels.PortainereeEndpoint) AccessGroup {
	environmentIds := make([]int, 0)
	for _, env := range rawEndpoints {
		if env.GroupID == rawGroup.ID {
			environmentIds = append(environmentIds, int(env.ID))
		}
	}

	return AccessGroup{
		ID:             int(rawGroup.ID),
		Name:           rawGroup.Name,
		EnvironmentIds: environmentIds,
		UserAccesses:   convertAccesses(rawGroup.UserAccessPolicies),
		TeamAccesses:   convertAccesses(rawGroup.TeamAccessPolicies),
	}
}
