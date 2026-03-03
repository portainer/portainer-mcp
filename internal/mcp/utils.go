package mcp

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// parseAccessMap parses access entries from an array of objects and returns a map of ID to access level
func parseAccessMap(entries []any) (map[int]string, error) {
	accessMap := map[int]string{}

	for _, entry := range entries {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid access entry: %v", entry)
		}

		id, ok := entryMap["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid ID: %v", entryMap["id"])
		}
		if id != float64(int(id)) {
			return nil, fmt.Errorf("invalid ID: %v is not a whole number", id)
		}

		access, ok := entryMap["access"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid access: %v", entryMap["access"])
		}

		if !isValidAccessLevel(access) {
			return nil, fmt.Errorf("invalid access level: %s", access)
		}

		accessMap[int(id)] = access
	}

	return accessMap, nil
}

// parseKeyValueMap parses a slice of map[string]any into a map[string]string,
// expecting each map to have "key" and "value" string fields.
func parseKeyValueMap(items []any) (map[string]string, error) {
	resultMap := map[string]string{}

	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid item: %v", item)
		}

		key, ok := itemMap["key"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid key: %v", itemMap["key"])
		}

		value, ok := itemMap["value"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid value: %v", itemMap["value"])
		}

		resultMap[key] = value
	}

	return resultMap, nil
}

func isValidHTTPMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "HEAD"}
	return slices.Contains(validMethods, method)
}

// CreateMCPRequest creates a new MCP tool request with the given arguments
func CreateMCPRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

// applyDockerDefaultFilters injects sensible default query parameters for
// Docker API endpoints known to produce very large responses. Defaults are
// only applied when the caller has not already set the relevant parameter,
// so explicit values from the LLM are never overridden.
func applyDockerDefaultFilters(path string, queryParams map[string]string) {
	lower := strings.ToLower(path)

	// /containers/json returns verbose metadata per container. Default to 10.
	if (lower == "/containers/json" || strings.HasSuffix(lower, "/containers/json")) && queryParams["limit"] == "" {
		queryParams["limit"] = "10"
	}

	// /images/json can also be very large. No standard limit param, but
	// adding a filter for dangling=false keeps the list manageable.
	if (lower == "/images/json" || strings.HasSuffix(lower, "/images/json")) && queryParams["filters"] == "" {
		queryParams["filters"] = `{"dangling":["false"]}`
	}
}

// verboseContainerFields lists JSON keys that are stripped from /containers/json
// responses. These fields contain deeply nested metadata (network internals,
// mount propagation options, host-config details) that bloat the response far
// beyond what is useful for an LLM to reason about.
var verboseContainerFields = []string{
	"NetworkSettings",
	"HostConfig",
	"Mounts",
	"GraphDriver",
	"Labels",
	"ImageID",
}

// compactDockerResponse strips verbose fields from known large-response
// Docker API endpoints. If the response is not JSON or compaction fails,
// the original bytes are returned unchanged.
func compactDockerResponse(path string, body []byte) []byte {
	lower := strings.ToLower(path)

	if lower == "/containers/json" || strings.HasSuffix(lower, "/containers/json") {
		return compactContainerList(body)
	}

	return body
}

// compactContainerList parses a /containers/json array and removes verbose
// per-container fields, reducing a typical 88K response to ~10-15K.
func compactContainerList(body []byte) []byte {
	var containers []map[string]any
	if err := json.Unmarshal(body, &containers); err != nil {
		return body // not valid JSON array, return as-is
	}

	for i := range containers {
		for _, field := range verboseContainerFields {
			delete(containers[i], field)
		}
	}

	compacted, err := json.Marshal(containers)
	if err != nil {
		return body
	}
	return compacted
}

// isDangerousDockerPath returns true when the given Docker API path should be
// blocked regardless of caller intent.  The rules cover the most common RCE
// and privilege-escalation vectors exposed through the Docker socket.
//
// Blocked path patterns (matched anywhere in the path segment):
//   - /exec  – container exec provides unrestricted shell access (RCE)
//   - /build – image build executes arbitrary Dockerfile commands (RCE)
//   - /containers/create with a privileged or sensitive-mount body is handled
//     by the caller; the path itself is not blocked here
func isDangerousDockerPath(path string) bool {
	dangerousSegments := []string{
		"/exec",
		"/build",
	}
	lower := strings.ToLower(path)
	for _, seg := range dangerousSegments {
		if strings.Contains(lower, seg) {
			return true
		}
	}
	return false
}

// isPrivilegedDockerContainerCreate returns true when the request body for a
// containers/create call indicates a privileged container or a host bind-mount
// to a sensitive path (e.g. /, /etc, /var, /proc, /sys).
// This is a conservative string-based check – it does not parse JSON so it
// cannot be bypassed by unusual whitespace or field ordering.
func isPrivilegedDockerContainerCreate(path, body string) bool {
	if !strings.Contains(strings.ToLower(path), "/containers/create") {
		return false
	}
	if strings.Contains(body, `"Privileged":true`) || strings.Contains(body, `"Privileged": true`) {
		return true
	}
	sensitiveMounts := []string{`"/etc:`, `"/proc:`, `"/sys:`, `"/var:`, `"/host:`, `"/:`}
	for _, m := range sensitiveMounts {
		if strings.Contains(body, m) {
			return true
		}
	}
	return false
}

// isDangerousKubernetesPath returns true when the Kubernetes API path and
// method combination should be blocked.
//
// Rules:
//   - /secrets accessed with any non-GET method (write access to secrets is a
//     privilege-escalation and credential-theft vector)
func isDangerousKubernetesPath(path, method string) bool {
	if strings.Contains(strings.ToLower(path), "/secrets") && method != "GET" {
		return true
	}
	return false
}

// hasWatchQueryParam returns true when the provided query parameter map
// contains watch=true, which causes the Kubernetes API server to open an
// unbounded streaming response.
func hasWatchQueryParam(queryParams map[string]string) bool {
	v, ok := queryParams["watch"]
	return ok && strings.EqualFold(v, "true")
}
