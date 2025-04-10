package models

import (
	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/portainer/portainer-mcp/pkg/portainer/utils"
)

type Environment struct {
	ID           int            `json:"id"`
	Name         string         `json:"name"`
	Status       string         `json:"status"`
	Type         string         `json:"type"`
	TagIds       []int          `json:"tag_ids"`
	UserAccesses map[int]string `json:"user_accesses"`
	TeamAccesses map[int]string `json:"team_accesses"`
}

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

func ConvertEndpointToEnvironment(rawEndpoint *apimodels.PortainereeEndpoint) Environment {
	return Environment{
		ID:           int(rawEndpoint.ID),
		Name:         rawEndpoint.Name,
		Status:       convertEnvironmentStatus(rawEndpoint),
		Type:         convertEnvironmentType(rawEndpoint),
		TagIds:       utils.Int64ToIntSlice(rawEndpoint.TagIds),
		UserAccesses: convertAccesses(rawEndpoint.UserAccessPolicies),
		TeamAccesses: convertAccesses(rawEndpoint.TeamAccessPolicies),
	}
}

func convertEnvironmentStatus(rawEndpoint *apimodels.PortainereeEndpoint) string {
	if rawEndpoint.Type == 4 || rawEndpoint.Type == 7 {
		return convertEdgeEnvironmentStatus(rawEndpoint)
	}
	return convertStandardEnvironmentStatus(rawEndpoint)
}

func convertStandardEnvironmentStatus(rawEndpoint *apimodels.PortainereeEndpoint) string {
	switch rawEndpoint.Status {
	case 1:
		return EnvironmentStatusActive
	case 2:
		return EnvironmentStatusInactive
	default:
		return EnvironmentStatusUnknown
	}
}

func convertEdgeEnvironmentStatus(rawEndpoint *apimodels.PortainereeEndpoint) string {
	if rawEndpoint.Heartbeat {
		return EnvironmentStatusActive
	}
	return EnvironmentStatusInactive
}

func convertEnvironmentType(rawEndpoint *apimodels.PortainereeEndpoint) string {
	switch rawEndpoint.Type {
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
