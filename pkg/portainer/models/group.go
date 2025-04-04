package models

import (
	"github.com/deviantony/portainer-mcp/pkg/portainer/utils"
	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
)

type Group struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	EnvironmentIds []int  `json:"environment_ids"`
	TagIds         []int  `json:"tag_ids"`
}

func ConvertEdgeGroupToGroup(rawEdgeGroup *apimodels.EdgegroupsDecoratedEdgeGroup) Group {
	return Group{
		ID:             int(rawEdgeGroup.ID),
		Name:           rawEdgeGroup.Name,
		EnvironmentIds: utils.Int64ToIntSlice(rawEdgeGroup.Endpoints),
		TagIds:         utils.Int64ToIntSlice(rawEdgeGroup.TagIds),
	}
}
