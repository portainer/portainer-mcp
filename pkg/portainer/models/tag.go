package models

import (
	"strconv"

	"github.com/portainer/client-api-go/v2/pkg/models"
)

type (
	EnvironmentTag struct {
		ID             int    `json:"id"`
		Name           string `json:"name"`
		EnvironmentIds []int  `json:"environment_ids"`
	}
)

func ConvertTagToEnvironmentTag(e *models.PortainerTag) EnvironmentTag {
	environmentIDs := make([]int, 0, len(e.Endpoints))

	for endpointID := range e.Endpoints {
		id, err := strconv.Atoi(endpointID)
		if err == nil {
			environmentIDs = append(environmentIDs, id)
		}
	}

	return EnvironmentTag{
		ID:             int(e.ID),
		Name:           e.Name,
		EnvironmentIds: environmentIDs,
	}
}
