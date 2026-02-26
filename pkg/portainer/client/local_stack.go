package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/portainer/portainer-mcp/pkg/portainer/models"
)

// apiRequest performs a raw HTTP request against the Portainer API.
func (c *rawHTTPClient) apiRequest(method, path string, body interface{}) (*http.Response, error) {
	url := c.serverURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpCli.Do(req)
}

// GetLocalStacks retrieves all regular (non-edge) stacks from the Portainer server.
//
// Returns:
//   - A slice of LocalStack objects
//   - An error if the operation fails
func (c *PortainerClient) GetLocalStacks() ([]models.LocalStack, error) {
	resp, err := c.rawCli.apiRequest(http.MethodGet, "/api/stacks", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list local stacks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list local stacks (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var rawStacks []models.RawLocalStack
	if err := json.NewDecoder(resp.Body).Decode(&rawStacks); err != nil {
		return nil, fmt.Errorf("failed to decode local stacks response: %w", err)
	}

	stacks := make([]models.LocalStack, len(rawStacks))
	for i, raw := range rawStacks {
		stacks[i] = models.ConvertRawLocalStackToLocalStack(raw)
	}

	return stacks, nil
}

// GetLocalStackFile retrieves the compose file content of a regular stack.
//
// Parameters:
//   - id: The ID of the stack
//
// Returns:
//   - The compose file content as a string
//   - An error if the operation fails
func (c *PortainerClient) GetLocalStackFile(id int) (string, error) {
	resp, err := c.rawCli.apiRequest(http.MethodGet, fmt.Sprintf("/api/stacks/%d/file", id), nil)
	if err != nil {
		return "", fmt.Errorf("failed to get local stack file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get local stack file (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var stackFile models.RawLocalStackFile
	if err := json.NewDecoder(resp.Body).Decode(&stackFile); err != nil {
		return "", fmt.Errorf("failed to decode local stack file response: %w", err)
	}

	return stackFile.StackFileContent, nil
}

// updateLocalStackRequest is the request body for updating a local stack
type updateLocalStackRequest struct {
	StackFileContent string                   `json:"stackFileContent"`
	Env              []models.LocalStackEnvVar `json:"env"`
	Prune            bool                     `json:"prune"`
	PullImage        bool                     `json:"pullImage"`
}

// UpdateLocalStack updates a regular stack's compose file and environment variables.
//
// Parameters:
//   - id: The stack ID
//   - endpointId: The environment/endpoint ID where the stack is deployed
//   - file: The new compose file content
//   - env: Environment variables for the stack
//   - prune: If true, services removed from the compose file will be stopped and removed
//   - pullImage: If true, images will be pulled before deploying
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateLocalStack(id, endpointId int, file string, env []models.LocalStackEnvVar, prune, pullImage bool) error {
	body := updateLocalStackRequest{
		StackFileContent: file,
		Env:              env,
		Prune:            prune,
		PullImage:        pullImage,
	}

	resp, err := c.rawCli.apiRequest(http.MethodPut, fmt.Sprintf("/api/stacks/%d?endpointId=%d", id, endpointId), body)
	if err != nil {
		return fmt.Errorf("failed to update local stack: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update local stack (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// createLocalStackRequest is the request body for creating a local stack
type createLocalStackRequest struct {
	Name             string                   `json:"name"`
	StackFileContent string                   `json:"stackFileContent"`
	Env              []models.LocalStackEnvVar `json:"env,omitempty"`
}

// CreateLocalStack creates a new standalone Docker Compose stack.
//
// Parameters:
//   - endpointId: The environment/endpoint ID where the stack will be deployed
//   - name: The name of the stack
//   - file: The compose file content
//   - env: Environment variables for the stack
//
// Returns:
//   - The ID of the created stack
//   - An error if the operation fails
func (c *PortainerClient) CreateLocalStack(endpointId int, name, file string, env []models.LocalStackEnvVar) (int, error) {
	body := createLocalStackRequest{
		Name:             name,
		StackFileContent: file,
		Env:              env,
	}

	resp, err := c.rawCli.apiRequest(http.MethodPost, fmt.Sprintf("/api/stacks/create/standalone/string?endpointId=%d", endpointId), body)
	if err != nil {
		return 0, fmt.Errorf("failed to create local stack: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to create local stack (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var raw models.RawLocalStack
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return 0, fmt.Errorf("failed to decode create stack response: %w", err)
	}

	return raw.ID, nil
}

// StartLocalStack starts a stopped local stack.
//
// Parameters:
//   - id: The stack ID
//   - endpointId: The environment/endpoint ID
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) StartLocalStack(id, endpointId int) error {
	resp, err := c.rawCli.apiRequest(http.MethodPost, fmt.Sprintf("/api/stacks/%d/start?endpointId=%d", id, endpointId), nil)
	if err != nil {
		return fmt.Errorf("failed to start local stack: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to start local stack (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// StopLocalStack stops a running local stack.
//
// Parameters:
//   - id: The stack ID
//   - endpointId: The environment/endpoint ID
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) StopLocalStack(id, endpointId int) error {
	resp, err := c.rawCli.apiRequest(http.MethodPost, fmt.Sprintf("/api/stacks/%d/stop?endpointId=%d", id, endpointId), nil)
	if err != nil {
		return fmt.Errorf("failed to stop local stack: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop local stack (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// DeleteLocalStack deletes a local stack.
//
// Parameters:
//   - id: The stack ID
//   - endpointId: The environment/endpoint ID
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) DeleteLocalStack(id, endpointId int) error {
	resp, err := c.rawCli.apiRequest(http.MethodDelete, fmt.Sprintf("/api/stacks/%d?endpointId=%d", id, endpointId), nil)
	if err != nil {
		return fmt.Errorf("failed to delete local stack: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete local stack (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
