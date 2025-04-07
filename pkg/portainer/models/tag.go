package models

import (
	"strconv"

	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
)

type EnvironmentTag struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	EnvironmentIds []int  `json:"environment_ids"`
}

func ConvertTagToEnvironmentTag(rawTag *apimodels.PortainerTag) EnvironmentTag {
	environmentIDs := make([]int, 0, len(rawTag.Endpoints))

	for endpointID := range rawTag.Endpoints {
		id, err := strconv.Atoi(endpointID)
		if err == nil {
			environmentIDs = append(environmentIDs, id)
		}
	}

	return EnvironmentTag{
		ID:             int(rawTag.ID),
		Name:           rawTag.Name,
		EnvironmentIds: environmentIDs,
	}
}
