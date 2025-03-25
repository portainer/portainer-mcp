package models

import (
	"reflect"
	"testing"

	"github.com/portainer/client-api-go/v2/pkg/models"
)

func TestConvertEndpointGroupToAccessGroup(t *testing.T) {
	tests := []struct {
		name     string
		group    *models.PortainerEndpointGroup
		envs     []*models.PortainereeEndpoint
		expected AccessGroup
	}{
		{
			name: "group with multiple environments and accesses",
			group: &models.PortainerEndpointGroup{
				ID:   1,
				Name: "Production",
				UserAccessPolicies: map[string]models.PortainerAccessPolicy{
					"1": {RoleID: 1},
					"2": {RoleID: 2},
				},
				TeamAccessPolicies: map[string]models.PortainerAccessPolicy{
					"10": {RoleID: 3},
					"20": {RoleID: 4},
				},
			},
			envs: []*models.PortainereeEndpoint{
				{ID: 100, GroupID: 1},
				{ID: 101, GroupID: 1},
				{ID: 102, GroupID: 2}, // Different group
			},
			expected: AccessGroup{
				ID:             1,
				Name:           "Production",
				EnvironmentIds: []int{100, 101},
				UserAccesses: map[int]string{
					1: "environment_administrator",
					2: "helpdesk_user",
				},
				TeamAccesses: map[int]string{
					10: "standard_user",
					20: "readonly_user",
				},
			},
		},
		{
			name: "group with no environments",
			group: &models.PortainerEndpointGroup{
				ID:   2,
				Name: "Empty",
				UserAccessPolicies: map[string]models.PortainerAccessPolicy{
					"1": {RoleID: 5},
				},
				TeamAccessPolicies: map[string]models.PortainerAccessPolicy{},
			},
			envs: []*models.PortainereeEndpoint{
				{ID: 100, GroupID: 1}, // Different group
			},
			expected: AccessGroup{
				ID:             2,
				Name:           "Empty",
				EnvironmentIds: []int{},
				UserAccesses: map[int]string{
					1: "operator_user",
				},
				TeamAccesses: map[int]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertEndpointGroupToAccessGroup(tt.group, tt.envs)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ConvertEndpointGroupToAccessGroup() = %v, want %v", result, tt.expected)
			}
		})
	}
}
