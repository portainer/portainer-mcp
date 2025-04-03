package client

import (
	"errors"
	"testing"

	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestGetSettings(t *testing.T) {
	tests := []struct {
		name          string
		mockSettings  *apimodels.PortainereeSettings
		mockError     error
		expected      models.PortainerSettings
		expectedError bool
	}{
		{
			name: "successful retrieval - internal auth",
			mockSettings: &apimodels.PortainereeSettings{
				AuthenticationMethod:      1, // internal
				EnableEdgeComputeFeatures: true,
				Edge: &apimodels.PortainereeEdge{
					TunnelServerAddress: "tunnel.example.com",
				},
			},
			expected: models.PortainerSettings{
				Authentication: struct {
					Method string `json:"method"`
				}{
					Method: models.AuthenticationMethodInternal,
				},
				Edge: struct {
					Enabled   bool   `json:"enabled"`
					ServerURL string `json:"server_url"`
				}{
					Enabled:   true,
					ServerURL: "tunnel.example.com",
				},
			},
		},
		{
			name: "successful retrieval - ldap auth",
			mockSettings: &apimodels.PortainereeSettings{
				AuthenticationMethod:      2, // ldap
				EnableEdgeComputeFeatures: false,
				Edge: &apimodels.PortainereeEdge{
					TunnelServerAddress: "tunnel2.example.com",
				},
			},
			expected: models.PortainerSettings{
				Authentication: struct {
					Method string `json:"method"`
				}{
					Method: models.AuthenticationMethodLDAP,
				},
				Edge: struct {
					Enabled   bool   `json:"enabled"`
					ServerURL string `json:"server_url"`
				}{
					Enabled:   false,
					ServerURL: "tunnel2.example.com",
				},
			},
		},
		{
			name: "successful retrieval - oauth auth",
			mockSettings: &apimodels.PortainereeSettings{
				AuthenticationMethod:      3, // oauth
				EnableEdgeComputeFeatures: true,
				Edge: &apimodels.PortainereeEdge{
					TunnelServerAddress: "tunnel3.example.com",
				},
			},
			expected: models.PortainerSettings{
				Authentication: struct {
					Method string `json:"method"`
				}{
					Method: models.AuthenticationMethodOAuth,
				},
				Edge: struct {
					Enabled   bool   `json:"enabled"`
					ServerURL string `json:"server_url"`
				}{
					Enabled:   true,
					ServerURL: "tunnel3.example.com",
				},
			},
		},
		{
			name: "successful retrieval - unknown auth",
			mockSettings: &apimodels.PortainereeSettings{
				AuthenticationMethod:      0, // unknown
				EnableEdgeComputeFeatures: false,
				Edge: &apimodels.PortainereeEdge{
					TunnelServerAddress: "tunnel4.example.com",
				},
			},
			expected: models.PortainerSettings{
				Authentication: struct {
					Method string `json:"method"`
				}{
					Method: models.AuthenticationMethodUnknown,
				},
				Edge: struct {
					Enabled   bool   `json:"enabled"`
					ServerURL string `json:"server_url"`
				}{
					Enabled:   false,
					ServerURL: "tunnel4.example.com",
				},
			},
		},
		{
			name:          "get settings error",
			mockError:     errors.New("failed to get settings"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockPortainerAPI)
			mockAPI.On("GetSettings").Return(tt.mockSettings, tt.mockError)

			client := &PortainerClient{cli: mockAPI}

			settings, err := client.GetSettings()

			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, settings)
			mockAPI.AssertExpectations(t)
		})
	}
}
