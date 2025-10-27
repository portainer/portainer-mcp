package models

import (
	"time"

	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/portainer/portainer-mcp/pkg/portainer/utils"
)

type Stack struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	CreatedAt           string `json:"created_at"`
	EnvironmentGroupIds []int  `json:"group_ids"`
}

// RegularStack represents a regular Docker stack from Portainer API
type RegularStack struct {
	ID           int    `json:"Id"`
	Name         string `json:"Name"`
	Type         int    `json:"Type"`
	EndpointId   int    `json:"EndpointId"`
	CreationDate int64  `json:"CreationDate"`
	Status       int    `json:"Status"`
}

func ConvertEdgeStackToStack(rawEdgeStack *apimodels.PortainereeEdgeStack) Stack {
	createdAt := time.Unix(rawEdgeStack.CreationDate, 0).Format(time.RFC3339)

	return Stack{
		ID:                  int(rawEdgeStack.ID),
		Name:                rawEdgeStack.Name,
		CreatedAt:           createdAt,
		EnvironmentGroupIds: utils.Int64ToIntSlice(rawEdgeStack.EdgeGroups),
	}
}

func ConvertRegularStackToStack(rawStack *RegularStack) Stack {
	createdAt := time.Unix(rawStack.CreationDate, 0).Format(time.RFC3339)

	return Stack{
		ID:                  rawStack.ID,
		Name:                rawStack.Name,
		CreatedAt:           createdAt,
		EnvironmentGroupIds: []int{}, // Regular stacks don't have group IDs, they have EndpointId
	}
}
