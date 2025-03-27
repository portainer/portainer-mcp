package mcp

import "testing"

func TestIsValidAccessLevel(t *testing.T) {
	tests := []struct {
		name        string
		accessLevel string
		want        bool
	}{
		{"ValidEnvironmentAdmin", AccessLevelEnvironmentAdmin, true},
		{"ValidHelpdeskUser", AccessLevelHelpdeskUser, true},
		{"ValidStandardUser", AccessLevelStandardUser, true},
		{"ValidReadonlyUser", AccessLevelReadonlyUser, true},
		{"ValidOperatorUser", AccessLevelOperatorUser, true},
		{"InvalidEmpty", "", false},
		{"InvalidRandom", "invalid_access", false},
		{"CaseSensitive", "ENVIRONMENT_ADMINISTRATOR", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidAccessLevel(tt.accessLevel); got != tt.want {
				t.Errorf("isValidAccessLevel(%q) = %v, want %v", tt.accessLevel, got, tt.want)
			}
		})
	}
}

func TestIsValidUserRole(t *testing.T) {
	tests := []struct {
		name     string
		userRole string
		want     bool
	}{
		{"ValidAdmin", UserRoleAdmin, true},
		{"ValidUser", UserRoleUser, true},
		{"ValidEdgeAdmin", UserRoleEdgeAdmin, true},
		{"InvalidEmpty", "", false},
		{"InvalidRandom", "invalid_role", false},
		{"CaseSensitive", "ADMIN", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidUserRole(tt.userRole); got != tt.want {
				t.Errorf("isValidUserRole(%q) = %v, want %v", tt.userRole, got, tt.want)
			}
		})
	}
}
