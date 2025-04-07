package models

import apimodels "github.com/portainer/client-api-go/v2/pkg/models"

type PortainerSettings struct {
	Authentication struct {
		Method string `json:"method"`
	} `json:"authentication"`
	Edge struct {
		Enabled   bool   `json:"enabled"`
		ServerURL string `json:"server_url"`
	} `json:"edge"`
}

const (
	AuthenticationMethodInternal = "internal"
	AuthenticationMethodLDAP     = "ldap"
	AuthenticationMethodOAuth    = "oauth"
	AuthenticationMethodUnknown  = "unknown"
)

func ConvertSettingsToPortainerSettings(rawSettings *apimodels.PortainereeSettings) PortainerSettings {
	s := PortainerSettings{}

	s.Authentication.Method = convertAuthenticationMethod(rawSettings.AuthenticationMethod)
	s.Edge.Enabled = rawSettings.EnableEdgeComputeFeatures
	s.Edge.ServerURL = rawSettings.Edge.TunnelServerAddress

	return s
}

func convertAuthenticationMethod(method int64) string {
	switch method {
	case 1:
		return AuthenticationMethodInternal
	case 2:
		return AuthenticationMethodLDAP
	case 3:
		return AuthenticationMethodOAuth
	default:
		return AuthenticationMethodUnknown
	}
}
