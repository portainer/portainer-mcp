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
			name: "active docker-local environment",
			endpoint: &models.PortainereeEndpoint{
				ID:     1,
				Name:   "local-docker",
				Status: 1, // active
				Type:   1, // docker-local
				TagIds: []int64{1, 2},
			},
			want: Environment{
				ID:     1,
				Name:   "local-docker",
				Status: EnvironmentStatusActive,
				Type:   EnvironmentTypeDockerLocal,
				TagIds: []int{1, 2},
			},
		},
		{
			name: "inactive kubernetes-agent environment",
			endpoint: &models.PortainereeEndpoint{
				ID:     2,
				Name:   "k8s-agent",
				Status: 2, // inactive
				Type:   7, // kubernetes-edge-agent
				TagIds: []int64{1},
			},
			want: Environment{
				ID:     2,
				Name:   "k8s-agent",
				Status: EnvironmentStatusInactive,
				Type:   EnvironmentTypeKubernetesEdgeAgent,
				TagIds: []int{1},
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
		name   string
		status int
		want   string
	}{
		{
			name:   "active status",
			status: 1,
			want:   EnvironmentStatusActive,
		},
		{
			name:   "inactive status",
			status: 2,
			want:   EnvironmentStatusInactive,
		},
		{
			name:   "unknown status",
			status: 0,
			want:   EnvironmentStatusUnknown,
		},
		{
			name:   "invalid status",
			status: 99,
			want:   EnvironmentStatusUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &models.PortainereeEndpoint{Status: int64(tt.status)}
			got := convertEnvironmentStatus(endpoint)
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
