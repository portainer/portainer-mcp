package models

import (
	"reflect"
	"testing"

	"github.com/portainer/client-api-go/v2/pkg/models"
)

func TestConvertEndpointToEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *models.PortainereeEndpoint
		want     Environment
	}{
		{
			name: "active docker-local environment with accesses",
			endpoint: &models.PortainereeEndpoint{
				ID:     1,
				Name:   "local-docker",
				Status: 1, // active
				Type:   1, // docker-local
				TagIds: []int64{1, 2},
				UserAccessPolicies: models.PortainerUserAccessPolicies{
					"1": models.PortainerAccessPolicy{RoleID: 1},
					"2": models.PortainerAccessPolicy{RoleID: 3},
				},
				TeamAccessPolicies: models.PortainerTeamAccessPolicies{
					"10": models.PortainerAccessPolicy{RoleID: 2},
					"20": models.PortainerAccessPolicy{RoleID: 4},
				},
			},
			want: Environment{
				ID:     1,
				Name:   "local-docker",
				Status: EnvironmentStatusActive,
				Type:   EnvironmentTypeDockerLocal,
				TagIds: []int{1, 2},
				UserAccesses: map[int]string{
					1: "environment_administrator",
					2: "standard_user",
				},
				TeamAccesses: map[int]string{
					10: "helpdesk_user",
					20: "readonly_user",
				},
			},
		},
		{
			name: "inactive kubernetes-agent environment with empty accesses",
			endpoint: &models.PortainereeEndpoint{
				ID:                 2,
				Name:               "k8s-agent",
				Status:             2, // inactive
				Type:               7, // kubernetes-edge-agent
				TagIds:             []int64{1},
				UserAccessPolicies: models.PortainerUserAccessPolicies{},
				TeamAccessPolicies: models.PortainerTeamAccessPolicies{},
			},
			want: Environment{
				ID:           2,
				Name:         "k8s-agent",
				Status:       EnvironmentStatusInactive,
				Type:         EnvironmentTypeKubernetesEdgeAgent,
				TagIds:       []int{1},
				UserAccesses: map[int]string{},
				TeamAccesses: map[int]string{},
			},
		},
		{
			name: "environment with invalid access IDs",
			endpoint: &models.PortainereeEndpoint{
				ID:     3,
				Name:   "invalid-access",
				Status: 1,
				Type:   1,
				TagIds: []int64{},
				UserAccessPolicies: models.PortainerUserAccessPolicies{
					"invalid": models.PortainerAccessPolicy{RoleID: 1},
					"2":       models.PortainerAccessPolicy{RoleID: 3},
				},
				TeamAccessPolicies: models.PortainerTeamAccessPolicies{
					"bad": models.PortainerAccessPolicy{RoleID: 2},
					"20":  models.PortainerAccessPolicy{RoleID: 4},
				},
			},
			want: Environment{
				ID:     3,
				Name:   "invalid-access",
				Status: EnvironmentStatusActive,
				Type:   EnvironmentTypeDockerLocal,
				TagIds: []int{},
				UserAccesses: map[int]string{
					2: "standard_user",
				},
				TeamAccesses: map[int]string{
					20: "readonly_user",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertEndpointToEnvironment(tt.endpoint)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertEndpointToEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertEnvironmentStatus(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *models.PortainereeEndpoint
		want     string
	}{
		{
			name: "standard environment - active status",
			endpoint: &models.PortainereeEndpoint{
				Status: 1,
				Type:   1, // docker-local
			},
			want: EnvironmentStatusActive,
		},
		{
			name: "standard environment - inactive status",
			endpoint: &models.PortainereeEndpoint{
				Status: 2,
				Type:   2, // docker-agent
			},
			want: EnvironmentStatusInactive,
		},
		{
			name: "standard environment - unknown status",
			endpoint: &models.PortainereeEndpoint{
				Status: 0,
				Type:   3, // azure-aci
			},
			want: EnvironmentStatusUnknown,
		},
		{
			name: "edge environment - active with heartbeat",
			endpoint: &models.PortainereeEndpoint{
				Type:      4, // docker-edge-agent
				Heartbeat: true,
			},
			want: EnvironmentStatusActive,
		},
		{
			name: "edge environment - inactive without heartbeat",
			endpoint: &models.PortainereeEndpoint{
				Type:      7, // kubernetes-edge-agent
				Heartbeat: false,
			},
			want: EnvironmentStatusInactive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertEnvironmentStatus(tt.endpoint)
			if got != tt.want {
				t.Errorf("convertEnvironmentStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertEnvironmentType(t *testing.T) {
	tests := []struct {
		name      string
		typeValue int
		want      string
	}{
		{
			name:      "docker-local type",
			typeValue: 1,
			want:      EnvironmentTypeDockerLocal,
		},
		{
			name:      "docker-agent type",
			typeValue: 2,
			want:      EnvironmentTypeDockerAgent,
		},
		{
			name:      "azure-aci type",
			typeValue: 3,
			want:      EnvironmentTypeAzureACI,
		},
		{
			name:      "docker-edge-agent type",
			typeValue: 4,
			want:      EnvironmentTypeDockerEdgeAgent,
		},
		{
			name:      "kubernetes-local type",
			typeValue: 5,
			want:      EnvironmentTypeKubernetesLocal,
		},
		{
			name:      "kubernetes-agent type",
			typeValue: 6,
			want:      EnvironmentTypeKubernetesAgent,
		},
		{
			name:      "kubernetes-edge-agent type",
			typeValue: 7,
			want:      EnvironmentTypeKubernetesEdgeAgent,
		},
		{
			name:      "unknown type",
			typeValue: 0,
			want:      EnvironmentTypeUnknown,
		},
		{
			name:      "invalid type",
			typeValue: 99,
			want:      EnvironmentTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &models.PortainereeEndpoint{Type: int64(tt.typeValue)}
			got := convertEnvironmentType(endpoint)
			if got != tt.want {
				t.Errorf("convertEnvironmentType() = %v, want %v", got, tt.want)
			}
		})
	}
}
