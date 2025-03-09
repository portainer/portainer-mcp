package client

import (
	"github.com/portainer/client-api-go/v2/client"
)

// PortainerClient is a wrapper around the Portainer SDK client
// that provides simplified access to Portainer API functionality.
type PortainerClient struct {
	cli *client.PortainerClient
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
