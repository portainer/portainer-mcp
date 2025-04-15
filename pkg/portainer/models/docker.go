package models

import "io"

// DockerProxyRequestOptions represents the options for a Docker API request to a specific Portainer environment.
type DockerProxyRequestOptions struct {
	// EnvironmentID is the ID of the environment to proxy the request to.
	EnvironmentID int
	// Method is the HTTP method to use (GET, POST, PUT, DELETE, etc.).
	Method string
	// Path is the Docker API endpoint path to proxy to (e.g., "/containers/json"). Must include the leading slash.
	Path string
	// QueryParams is a map of query parameters to include in the request URL.
	QueryParams map[string]string
	// Headers is a map of headers to include in the request.
	Headers map[string]string
	// Body is the request body to send (set it to nil for requests that don't have a body).
	Body io.Reader
}
