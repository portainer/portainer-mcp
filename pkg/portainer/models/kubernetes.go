package models

import "io"

// KubernetesProxyRequestOptions represents the options for a Kubernetes API request to a specific Portainer environment.
type KubernetesProxyRequestOptions struct {
	// EnvironmentID is the ID of the environment to proxy the request to.
	EnvironmentID int
	// Method is the HTTP method to use (GET, POST, PUT, DELETE, etc.).
	Method string
	// Path is the Kubernetes API endpoint path to proxy to (e.g., "/api/v1/namespaces/default/pods"). Must include the leading slash.
	Path string
	// QueryParams is a map of query parameters to include in the request URL.
	QueryParams map[string]string
	// Headers is a map of headers to include in the request.
	Headers map[string]string
	// Body is the request body to send (set it to nil for requests that don't have a body).
	Body io.Reader
}
