package models

import (
	"testing"

	"github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestConvertAuthenticationMethod(t *testing.T) {
	tests := []struct {
		name           string
		methodID       int64
		expectedMethod string
	}{
		{
			name:           "Internal authentication",
			methodID:       1,
			expectedMethod: AuthenticationMethodInternal,
		},
		{
			name:           "LDAP authentication",
			methodID:       2,
			expectedMethod: AuthenticationMethodLDAP,
		},
		{
			name:           "OAuth authentication",
			methodID:       3,
			expectedMethod: AuthenticationMethodOAuth,
		},
		{
			name:           "Unknown authentication (0)",
			methodID:       0,
			expectedMethod: AuthenticationMethodUnknown,
		},
		{
			name:           "Unknown authentication (negative)",
			methodID:       -1,
			expectedMethod: AuthenticationMethodUnknown,
		},
		{
			name:           "Unknown authentication (large value)",
			methodID:       999,
			expectedMethod: AuthenticationMethodUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertAuthenticationMethod(tt.methodID)
			assert.Equal(t, tt.expectedMethod, result)
		})
	}
}

func TestConvertSettingsToPortainerSettings(t *testing.T) {
	tests := []struct {
		name           string
		input          *models.PortainereeSettings
		expectedOutput PortainerSettings
		shouldPanic    bool
	}{
		{
			name: "Complete settings conversion",
			input: &models.PortainereeSettings{
				AuthenticationMethod:      1,
				EnableEdgeComputeFeatures: true,
				Edge: &models.PortainereeEdge{
					TunnelServerAddress: "https://edge.example.com",
				},
			},
			expectedOutput: PortainerSettings{
				Authentication: struct {
					Method string `json:"method"`
				}{
					Method: AuthenticationMethodInternal,
				},
				Edge: struct {
					Enabled   bool   `json:"enabled"`
					ServerURL string `json:"server_url"`
				}{
					Enabled:   true,
					ServerURL: "https://edge.example.com",
				},
			},
		},
		{
			name: "Settings with LDAP authentication",
			input: &models.PortainereeSettings{
				AuthenticationMethod:      2,
				EnableEdgeComputeFeatures: false,
				Edge: &models.PortainereeEdge{
					TunnelServerAddress: "",
				},
			},
			expectedOutput: PortainerSettings{
				Authentication: struct {
					Method string `json:"method"`
				}{
					Method: AuthenticationMethodLDAP,
				},
				Edge: struct {
					Enabled   bool   `json:"enabled"`
					ServerURL string `json:"server_url"`
				}{
					Enabled:   false,
					ServerURL: "",
				},
			},
		},
		{
			name: "Settings with OAuth authentication",
			input: &models.PortainereeSettings{
				AuthenticationMethod:      3,
				EnableEdgeComputeFeatures: true,
				Edge: &models.PortainereeEdge{
					TunnelServerAddress: "https://tunnel.portainer.io",
				},
			},
			expectedOutput: PortainerSettings{
				Authentication: struct {
					Method string `json:"method"`
				}{
					Method: AuthenticationMethodOAuth,
				},
				Edge: struct {
					Enabled   bool   `json:"enabled"`
					ServerURL string `json:"server_url"`
				}{
					Enabled:   true,
					ServerURL: "https://tunnel.portainer.io",
				},
			},
		},
		{
			name: "Settings with unknown authentication",
			input: &models.PortainereeSettings{
				AuthenticationMethod:      99,
				EnableEdgeComputeFeatures: false,
				Edge: &models.PortainereeEdge{
					TunnelServerAddress: "",
				},
			},
			expectedOutput: PortainerSettings{
				Authentication: struct {
					Method string `json:"method"`
				}{
					Method: AuthenticationMethodUnknown,
				},
				Edge: struct {
					Enabled   bool   `json:"enabled"`
					ServerURL string `json:"server_url"`
				}{
					Enabled:   false,
					ServerURL: "",
				},
			},
		},
		{
			name:        "Nil input",
			input:       nil,
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.Panics(t, func() {
					ConvertSettingsToPortainerSettings(tt.input)
				})
				return
			}

			result := ConvertSettingsToPortainerSettings(tt.input)
			assert.Equal(t, tt.expectedOutput.Authentication.Method, result.Authentication.Method)
			assert.Equal(t, tt.expectedOutput.Edge.Enabled, result.Edge.Enabled)
			assert.Equal(t, tt.expectedOutput.Edge.ServerURL, result.Edge.ServerURL)
		})
	}
}
