package models

import (
	"strconv"

	"github.com/portainer/client-api-go/v2/pkg/models"
)

func convertAccesses[T models.PortainerUserAccessPolicies | models.PortainerTeamAccessPolicies](policies T) map[int]string {
	accesses := make(map[int]string)
	for idStr, role := range policies {
		id, err := strconv.Atoi(idStr)
		if err == nil {
			accesses[id] = convertAccessPolicyRole(&role)
		}
	}
	return accesses
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
