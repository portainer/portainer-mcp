package models

import (
	"strconv"

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
		UserAccesses:   convertUserAccesses(g),
		TeamAccesses:   convertTeamAccesses(g),
	}
}

func convertUserAccesses(g *models.PortainerEndpointGroup) map[int]string {
	userAccesses := make(map[int]string)
	for userID, role := range g.UserAccessPolicies {
		id, err := strconv.Atoi(userID)
		if err == nil {
			userAccesses[id] = convertAccessPolicyRole(&role)
		}
	}
	return userAccesses
}

func convertTeamAccesses(g *models.PortainerEndpointGroup) map[int]string {
	teamAccesses := make(map[int]string)
	for teamID, role := range g.TeamAccessPolicies {
		id, err := strconv.Atoi(teamID)
		if err == nil {
			teamAccesses[id] = convertAccessPolicyRole(&role)
		}
	}
	return teamAccesses
}

func convertAccessPolicyRole(role *models.PortainerAccessPolicy) string {
	switch role.RoleID {
	case 1:
		return "environment_administrator"
	case 2:
		return "helpdesk_user"
	case 3:
		return "standard_user"
	case 4:
		return "readonly_user"
	case 5:
		return "operator_user"
	default:
		return "unknown"
	}
}
