package models

import (
	"time"

	"github.com/deviantony/mcp-go/pkg/portainer/utils"
	"github.com/portainer/client-api-go/v2/pkg/models"
)

type (
	Stack struct {
		ID                  int    `json:"id"`
		Name                string `json:"name"`
		CreatedAt           string `json:"created_at"`
		EnvironmentGroupIds []int  `json:"group_ids"`
	}
)

func ConvertEdgeStackToStack(e *models.PortainereeEdgeStack) Stack {
	createdAt := time.Unix(e.CreationDate, 0).Format(time.RFC3339)

	return Stack{
		ID:                  int(e.ID),
		Name:                e.Name,
		CreatedAt:           createdAt,
		EnvironmentGroupIds: utils.Int64ToIntSlice(e.EdgeGroups),
	}
}
