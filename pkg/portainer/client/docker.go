package client

import (
	"io"
	"net/http"
)

// ProxyDockerRequest proxies a Docker API request to a specific Portainer environment.
//
// Parameters:
//   - environmentId: The ID of the environment to proxy the request to
//   - dockerAPIPath: The path of the Docker API operation to proxy. Must include the leading slash. Example: /containers/json
//   - method: The HTTP method to use for the request
//   - body: The body of the request. Can be set to nil for requests that do not have a body.
func (c *PortainerClient) ProxyDockerRequest(environmentId int, dockerAPIPath string, method string, body io.Reader) (*http.Response, error) {
	return c.cli.ProxyDockerRequest(environmentId, dockerAPIPath, method, body)
}
