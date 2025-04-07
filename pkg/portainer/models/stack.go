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

func ConvertEdgeStackToStack(rawEdgeStack *apimodels.PortainereeEdgeStack) Stack {
	createdAt := time.Unix(rawEdgeStack.CreationDate, 0).Format(time.RFC3339)

	return Stack{
		ID:                  int(rawEdgeStack.ID),
		Name:                rawEdgeStack.Name,
		CreatedAt:           createdAt,
		EnvironmentGroupIds: utils.Int64ToIntSlice(rawEdgeStack.EdgeGroups),
	}
}
