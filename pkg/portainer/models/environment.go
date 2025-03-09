package models

import (
	"github.com/deviantony/mcp-go/pkg/portainer/utils"
	"github.com/portainer/client-api-go/v2/pkg/models"
)

type (
	Environment struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
		Type   string `json:"type"`
		TagIds []int  `json:"tag_ids"`
	}
)

// Environment status constants
const (
	EnvironmentStatusActive   = "active"
	EnvironmentStatusInactive = "inactive"
	EnvironmentStatusUnknown  = "unknown"
)

// Environment type constants
const (
	EnvironmentTypeDockerLocal         = "docker-local"
	EnvironmentTypeDockerAgent         = "docker-agent"
	EnvironmentTypeAzureACI            = "azure-aci"
	EnvironmentTypeDockerEdgeAgent     = "docker-edge-agent"
	EnvironmentTypeKubernetesLocal     = "kubernetes-local"
	EnvironmentTypeKubernetesAgent     = "kubernetes-agent"
	EnvironmentTypeKubernetesEdgeAgent = "kubernetes-edge-agent"
	EnvironmentTypeUnknown             = "unknown"
)

func ConvertEndpointToEnvironment(e *models.PortainereeEndpoint) Environment {
	return Environment{
		ID:     int(e.ID),
		Name:   e.Name,
		Status: convertEnvironmentStatus(e),
		Type:   convertEnvironmentType(e),
		TagIds: utils.Int64ToIntSlice(e.TagIds),
	}
}

func convertEnvironmentStatus(e *models.PortainereeEndpoint) string {
	switch e.Status {
	case 1:
		return EnvironmentStatusActive
	case 2:
		return EnvironmentStatusInactive
	default:
		return EnvironmentStatusUnknown
	}
}

func convertEnvironmentType(e *models.PortainereeEndpoint) string {
	switch e.Type {
	case 1:
		return EnvironmentTypeDockerLocal
	case 2:
		return EnvironmentTypeDockerAgent
	case 3:
		return EnvironmentTypeAzureACI
	case 4:
		return EnvironmentTypeDockerEdgeAgent
	case 5:
		return EnvironmentTypeKubernetesLocal
	case 6:
		return EnvironmentTypeKubernetesAgent
	case 7:
		return EnvironmentTypeKubernetesEdgeAgent
	default:
		return EnvironmentTypeUnknown
	}
}
