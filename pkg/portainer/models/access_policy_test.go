package models

import (
	"reflect"
	"testing"

	"github.com/portainer/client-api-go/v2/pkg/models"
)

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

func TestConvertAccesses(t *testing.T) {
	t.Run("user accesses", func(t *testing.T) {
		policies := models.PortainerUserAccessPolicies{
			"1": models.PortainerAccessPolicy{RoleID: 1},
			"2": models.PortainerAccessPolicy{RoleID: 3},
		}
		expected := map[int]string{
			1: "environment_administrator",
			2: "standard_user",
		}
		result := convertAccesses(policies)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("convertAccesses() = %v, want %v", result, expected)
		}
	})

	t.Run("team accesses", func(t *testing.T) {
		policies := models.PortainerTeamAccessPolicies{
			"10": models.PortainerAccessPolicy{RoleID: 1},
			"20": models.PortainerAccessPolicy{RoleID: 4},
		}
		expected := map[int]string{
			10: "environment_administrator",
			20: "readonly_user",
		}
		result := convertAccesses(policies)
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("convertAccesses() = %v, want %v", result, expected)
		}
	})
}
