package client

import (
	"net/http"

	"github.com/portainer/client-api-go/v2/client"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
)

// ProxyKubernetesRequest proxies a Kubernetes API request to a specific Portainer environment.
//
// Parameters:
//   - opts: Options defining the proxied request (environmentID, method, path, query params, headers, body)
//
// Returns:
//   - *http.Response: The response from the Kubernetes API
//   - error: Any error that occurred during the request
func (c *PortainerClient) ProxyKubernetesRequest(opts models.KubernetesProxyRequestOptions) (*http.Response, error) {
	proxyOpts := client.ProxyRequestOptions{
		Method:  opts.Method,
		APIPath: opts.Path,
		Body:    opts.Body,
	}

	if len(opts.QueryParams) > 0 {
		proxyOpts.QueryParams = opts.QueryParams
	}

	if len(opts.Headers) > 0 {
		proxyOpts.Headers = opts.Headers
	}

	return c.cli.ProxyKubernetesRequest(opts.EnvironmentID, proxyOpts)
}
