package client

import (
	"net/http"

	"github.com/portainer/client-api-go/v2/client"
	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
)

// PortainerAPIClient defines the interface for the underlying Portainer API client
type PortainerAPIClient interface {
	ListEdgeGroups() ([]*apimodels.EdgegroupsDecoratedEdgeGroup, error)
	CreateEdgeGroup(name string, environmentIds []int64) (int64, error)
	UpdateEdgeGroup(id int64, name *string, environmentIds *[]int64, tagIds *[]int64) error
	ListEdgeStacks() ([]*apimodels.PortainereeEdgeStack, error)
	CreateEdgeStack(name string, file string, environmentGroupIds []int64) (int64, error)
	UpdateEdgeStack(id int64, file string, environmentGroupIds []int64) error
	GetEdgeStackFile(id int64) (string, error)
	ListEndpointGroups() ([]*apimodels.PortainerEndpointGroup, error)
	CreateEndpointGroup(name string, associatedEndpoints []int64) (int64, error)
	UpdateEndpointGroup(id int64, name *string, userAccesses *map[int64]string, teamAccesses *map[int64]string) error
	AddEnvironmentToEndpointGroup(groupId int64, environmentId int64) error
	RemoveEnvironmentFromEndpointGroup(groupId int64, environmentId int64) error
	ListEndpoints() ([]*apimodels.PortainereeEndpoint, error)
	GetEndpoint(id int64) (*apimodels.PortainereeEndpoint, error)
	UpdateEndpoint(id int64, tagIds *[]int64, userAccesses *map[int64]string, teamAccesses *map[int64]string) error
	GetSettings() (*apimodels.PortainereeSettings, error)
	ListTags() ([]*apimodels.PortainerTag, error)
	CreateTag(name string) (int64, error)
	ListTeams() ([]*apimodels.PortainerTeam, error)
	ListTeamMemberships() ([]*apimodels.PortainerTeamMembership, error)
	CreateTeam(name string) (int64, error)
	UpdateTeamName(id int, name string) error
	DeleteTeamMembership(id int) error
	CreateTeamMembership(teamId int, userId int) error
	ListUsers() ([]*apimodels.PortainereeUser, error)
	UpdateUserRole(id int, role int64) error
	GetVersion() (string, error)
	ProxyDockerRequest(environmentId int, opts client.ProxyRequestOptions) (*http.Response, error)
	ProxyKubernetesRequest(environmentId int, opts client.ProxyRequestOptions) (*http.Response, error)
}

// PortainerClient is a wrapper around the Portainer SDK client
// that provides simplified access to Portainer API functionality.
type PortainerClient struct {
	cli PortainerAPIClient
}

// ClientOption defines a function that configures a PortainerClient.
type ClientOption func(*clientOptions)

// clientOptions holds configuration options for the PortainerClient.
type clientOptions struct {
	skipTLSVerify bool
}

// WithSkipTLSVerify configures whether to skip TLS certificate verification.
// Setting this to true is not recommended for production environments.
func WithSkipTLSVerify(skip bool) ClientOption {
	return func(o *clientOptions) {
		o.skipTLSVerify = skip
	}
}

// NewPortainerClient creates a new PortainerClient instance with the provided
// server URL and authentication token.
//
// Parameters:
//   - serverURL: The base URL of the Portainer server
//   - token: The authentication token for API access
//   - opts: Optional configuration options for the client
//
// Returns:
//   - A configured PortainerClient ready for API operations
func NewPortainerClient(serverURL string, token string, opts ...ClientOption) *PortainerClient {
	options := clientOptions{
		skipTLSVerify: false, // Default to secure TLS verification
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &PortainerClient{
		cli: client.NewPortainerClient(serverURL, token, client.WithSkipTLSVerify(options.skipTLSVerify)),
	}
}
