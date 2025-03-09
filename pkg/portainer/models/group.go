package models

import (
	"github.com/deviantony/mcp-go/pkg/portainer/utils"
	"github.com/portainer/client-api-go/v2/pkg/models"
)

type (
	Group struct {
		ID             int    `json:"id"`
		Name           string `json:"name"`
		EnvironmentIds []int  `json:"environment_ids"`
	}
)

func ConvertEdgeGroupToGroup(e *models.EdgegroupsDecoratedEdgeGroup) Group {
	return Group{
		ID:             int(e.ID),
		Name:           e.Name,
		EnvironmentIds: utils.Int64ToIntSlice(e.Endpoints),
	}
}
