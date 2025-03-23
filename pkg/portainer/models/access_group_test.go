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

func TestConvertAccessPolicyRole(t *testing.T) {
	tests := []struct {
		name     string
		role     *models.PortainerAccessPolicy
		expected string
	}{
		{
			name:     "environment administrator role",
			role:     &models.PortainerAccessPolicy{RoleID: 1},
			expected: "environment_administrator",
		},
		{
			name:     "helpdesk user role",
			role:     &models.PortainerAccessPolicy{RoleID: 2},
			expected: "helpdesk_user",
		},
		{
			name:     "standard user role",
			role:     &models.PortainerAccessPolicy{RoleID: 3},
			expected: "standard_user",
		},
		{
			name:     "readonly user role",
			role:     &models.PortainerAccessPolicy{RoleID: 4},
			expected: "readonly_user",
		},
		{
			name:     "operator user role",
			role:     &models.PortainerAccessPolicy{RoleID: 5},
			expected: "operator_user",
		},
		{
			name:     "unknown role",
			role:     &models.PortainerAccessPolicy{RoleID: 999},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertAccessPolicyRole(tt.role)
			if result != tt.expected {
				t.Errorf("convertAccessPolicyRole() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertUserAccesses(t *testing.T) {
	tests := []struct {
		name     string
		group    *models.PortainerEndpointGroup
		expected map[int]string
	}{
		{
			name: "multiple user accesses",
			group: &models.PortainerEndpointGroup{
				UserAccessPolicies: map[string]models.PortainerAccessPolicy{
					"1": {RoleID: 1},
					"2": {RoleID: 3},
				},
			},
			expected: map[int]string{
				1: "environment_administrator",
				2: "standard_user",
			},
		},
		{
			name: "invalid user ID",
			group: &models.PortainerEndpointGroup{
				UserAccessPolicies: map[string]models.PortainerAccessPolicy{
					"invalid": {RoleID: 1},
					"2":       {RoleID: 3},
				},
			},
			expected: map[int]string{
				2: "standard_user",
			},
		},
		{
			name: "empty user accesses",
			group: &models.PortainerEndpointGroup{
				UserAccessPolicies: map[string]models.PortainerAccessPolicy{},
			},
			expected: map[int]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertUserAccesses(tt.group)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertUserAccesses() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertTeamAccesses(t *testing.T) {
	tests := []struct {
		name     string
		group    *models.PortainerEndpointGroup
		expected map[int]string
	}{
		{
			name: "multiple team accesses",
			group: &models.PortainerEndpointGroup{
				TeamAccessPolicies: map[string]models.PortainerAccessPolicy{
					"10": {RoleID: 1},
					"20": {RoleID: 4},
				},
			},
			expected: map[int]string{
				10: "environment_administrator",
				20: "readonly_user",
			},
		},
		{
			name: "invalid team ID",
			group: &models.PortainerEndpointGroup{
				TeamAccessPolicies: map[string]models.PortainerAccessPolicy{
					"invalid": {RoleID: 1},
					"20":      {RoleID: 4},
				},
			},
			expected: map[int]string{
				20: "readonly_user",
			},
		},
		{
			name: "empty team accesses",
			group: &models.PortainerEndpointGroup{
				TeamAccessPolicies: map[string]models.PortainerAccessPolicy{},
			},
			expected: map[int]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTeamAccesses(tt.group)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertTeamAccesses() = %v, want %v", result, tt.expected)
			}
		})
	}
}
