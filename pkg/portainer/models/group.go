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

func ConvertEdgeGroupToGroup(e *apimodels.EdgegroupsDecoratedEdgeGroup) Group {
	return Group{
		ID:             int(e.ID),
		Name:           e.Name,
		EnvironmentIds: utils.Int64ToIntSlice(e.Endpoints),
		TagIds:         utils.Int64ToIntSlice(e.TagIds),
	}
}
