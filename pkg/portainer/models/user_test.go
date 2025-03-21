package models

import (
	"testing"

	"github.com/portainer/client-api-go/v2/pkg/models"
)

func TestConvertToUser(t *testing.T) {
	tests := []struct {
		name     string
		input    *models.PortainereeUser
		expected User
	}{
		{
			name: "admin user",
			input: &models.PortainereeUser{
				ID:       1,
				Username: "admin",
				Role:     1,
			},
			expected: User{
				ID:       1,
				Username: "admin",
				Role:     UserRoleAdmin,
			},
		},
		{
			name: "regular user",
			input: &models.PortainereeUser{
				ID:       2,
				Username: "user1",
				Role:     2,
			},
			expected: User{
				ID:       2,
				Username: "user1",
				Role:     UserRoleUser,
			},
		},
		{
			name: "edge admin user",
			input: &models.PortainereeUser{
				ID:       3,
				Username: "edge_admin",
				Role:     3,
			},
			expected: User{
				ID:       3,
				Username: "edge_admin",
				Role:     UserRoleEdgeAdmin,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToUser(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertToUser() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertUserRole(t *testing.T) {
	tests := []struct {
		name     string
		input    *models.PortainereeUser
		expected string
	}{
		{
			name:     "admin role",
			input:    &models.PortainereeUser{Role: 1},
			expected: UserRoleAdmin,
		},
		{
			name:     "user role",
			input:    &models.PortainereeUser{Role: 2},
			expected: UserRoleUser,
		},
		{
			name:     "edge admin role",
			input:    &models.PortainereeUser{Role: 3},
			expected: UserRoleEdgeAdmin,
		},
		{
			name:     "unknown role",
			input:    &models.PortainereeUser{Role: 999},
			expected: UserRoleUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertUserRole(tt.input)
			if result != tt.expected {
				t.Errorf("convertUserRole() = %v, want %v", result, tt.expected)
			}
		})
	}
}
