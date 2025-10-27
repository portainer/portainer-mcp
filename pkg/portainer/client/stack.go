package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/portainer/portainer-mcp/pkg/portainer/utils"
)

// GetStacks retrieves all stacks from the Portainer server.
// This function queries regular Docker stacks via the Portainer REST API.
// Falls back to edge stacks if regular stacks API fails.
//
// Returns:
//   - A slice of Stack objects
//   - An error if the operation fails
func (c *PortainerClient) GetStacks() ([]models.Stack, error) {
	// Try to get regular stacks first
	regularStacks, err := c.listRegularStacksHTTP()
	if err == nil && len(regularStacks) > 0 {
		stacks := make([]models.Stack, len(regularStacks))
		for i, rs := range regularStacks {
			stacks[i] = models.ConvertRegularStackToStack(&rs)
		}
		return stacks, nil
	}

	// Fallback to edge stacks if regular stacks failed
	edgeStacks, edgeErr := c.cli.ListEdgeStacks()
	if edgeErr != nil {
		// If both fail, return the original error from regular stacks
		if err != nil {
			return nil, fmt.Errorf("failed to list regular stacks: %w (edge stacks also failed: %v)", err, edgeErr)
		}
		return nil, fmt.Errorf("failed to list edge stacks: %w", edgeErr)
	}

	stacks := make([]models.Stack, len(edgeStacks))
	for i, es := range edgeStacks {
		stacks[i] = models.ConvertEdgeStackToStack(es)
	}

	return stacks, nil
}

// listRegularStacksHTTP makes a direct HTTP call to Portainer API to list regular Docker stacks
func (c *PortainerClient) listRegularStacksHTTP() ([]models.RegularStack, error) {
	// Build the API URL - add https:// if not present
	serverURL := c.serverURL
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "https://" + serverURL
	}
	apiURL := fmt.Sprintf("%s/api/stacks", strings.TrimSuffix(serverURL, "/"))

	// Create HTTP request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("X-API-Key", c.token)

	// Create HTTP client with TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.skipTLSVerify},
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var stacks []models.RegularStack
	if err := json.Unmarshal(body, &stacks); err != nil {
		return nil, fmt.Errorf("failed to parse response JSON: %w", err)
	}

	return stacks, nil
}

// GetStackFile retrieves the file content of a stack from the Portainer server.
// This function queries regular Docker stacks via the Portainer REST API.
// Falls back to edge stacks if regular stacks API fails.
//
// Parameters:
//   - id: The ID of the stack to retrieve
//
// Returns:
//   - The file content of the stack (Compose file)
//   - An error if the operation fails
func (c *PortainerClient) GetStackFile(id int) (string, error) {
	// Try to get regular stack file first
	file, err := c.getRegularStackFileHTTP(id)
	if err == nil && file != "" {
		return file, nil
	}

	// Fallback to edge stack file
	edgeFile, edgeErr := c.cli.GetEdgeStackFile(int64(id))
	if edgeErr != nil {
		// If both fail, return the original error from regular stacks
		if err != nil {
			return "", fmt.Errorf("failed to get regular stack file: %w (edge stack also failed: %v)", err, edgeErr)
		}
		return "", fmt.Errorf("failed to get edge stack file: %w", edgeErr)
	}

	return edgeFile, nil
}

// getRegularStackFileHTTP makes a direct HTTP call to Portainer API to get regular Docker stack file
func (c *PortainerClient) getRegularStackFileHTTP(id int) (string, error) {
	// Build the API URL
	serverURL := c.serverURL
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "https://" + serverURL
	}
	apiURL := fmt.Sprintf("%s/api/stacks/%d/file", strings.TrimSuffix(serverURL, "/"), id)

	// Create HTTP request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("X-API-Key", c.token)

	// Create HTTP client with TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.skipTLSVerify},
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response - Portainer returns {"StackFileContent": "yaml content"}
	var response struct {
		StackFileContent string `json:"StackFileContent"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response JSON: %w", err)
	}

	return response.StackFileContent, nil
}

// CreateStack creates a new stack on the Portainer server.
// This function specifically creates a Docker Compose stack.
// Stacks are the equivalent of Edge Stacks in Portainer.
//
// Parameters:
//   - name: The name of the stack
//   - file: The file content of the stack (Compose file)
//   - environmentGroupIds: A slice of environment group IDs to include in the stack
//
// Returns:
//   - The ID of the created stack
//   - An error if the operation fails
func (c *PortainerClient) CreateStack(name, file string, environmentGroupIds []int) (int, error) {
	id, err := c.cli.CreateEdgeStack(name, file, utils.IntToInt64Slice(environmentGroupIds))
	if err != nil {
		return 0, fmt.Errorf("failed to create edge stack: %w", err)
	}

	return int(id), nil
}

// UpdateStack updates an existing stack on the Portainer server.
// This function specifically updates a Docker Compose stack.
// Stacks are the equivalent of Edge Stacks in Portainer.
//
// Parameters:
//   - id: The ID of the stack to update
//   - file: The file content of the stack (Compose file)
//   - environmentGroupIds: A slice of environment group IDs to include in the stack
//
// Returns:
//   - An error if the operation fails
func (c *PortainerClient) UpdateStack(id int, file string, environmentGroupIds []int) error {
	err := c.cli.UpdateEdgeStack(int64(id), file, utils.IntToInt64Slice(environmentGroupIds))
	if err != nil {
		return fmt.Errorf("failed to update edge stack: %w", err)
	}

	return nil
}
