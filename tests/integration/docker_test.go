package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	mcpmodels "github.com/mark3labs/mcp-go/mcp"

	"github.com/portainer/portainer-mcp/internal/mcp"
	"github.com/portainer/portainer-mcp/tests/integration/containers"
	"github.com/portainer/portainer-mcp/tests/integration/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Test data constants
	testLocalEndpointName = "test-local-endpoint"
	testLocalEndpointID   = 1
	testVolumeName        = "test-proxy-volume"
)

// prepareDockerProxyTestEnvironment prepares the test environment for the tests
// It creates a local Docker endpoint
func prepareDockerProxyTestEnvironment(t *testing.T, env *helpers.TestEnv) {
	_, err := env.RawClient.CreateLocalDockerEndpoint(testLocalEndpointName)
	require.NoError(t, err, "Failed to create Local Docker endpoint")
}

// TestDockerProxy is an integration test suite that verifies the Docker proxy functionality
// provided by the Portainer MCP server. It tests the ability to proxy various Docker API requests
// to a specified environment, including:
// - Retrieving Docker version information (GET /version)
// - Creating a volume (POST /volumes/create)
// - Listing volumes with filters (GET /volumes?filters=...)
// - Removing a volume (DELETE /volumes/{name})
// It primarily tests against volumes, as testing container operations would require
// pulling images beforehand, potentially leading to rate limiting issues in CI/CD
// or rapid testing scenarios.
func TestDockerProxy(t *testing.T) {
	env := helpers.NewTestEnv(t, containers.WithDockerSocketBind(true))
	defer env.Cleanup(t)

	// Prepare the test environment
	prepareDockerProxyTestEnvironment(t, env)

	// Subtest: GET /version
	// Verifies that:
	// - A simple GET request to the Docker /version endpoint can be successfully proxied.
	// - The handler returns a non-empty response without errors.
	// - The response content contains expected fields like ApiVersion and Version.
	t.Run("GET /version", func(t *testing.T) {
		request := mcp.CreateMCPRequest(map[string]any{
			"environmentId": float64(testLocalEndpointID),
			"method":        "GET",
			"dockerAPIPath": "/version",
			"queryParams":   nil, // No query params for /version
			"headers":       nil, // No specific headers needed
			"body":          "",  // No body for GET request
		})

		handler := env.MCPServer.HandleDockerProxy()
		result, err := handler(env.Ctx, request)

		require.NoError(t, err, "Handler execution failed")
		require.NotNil(t, result, "Handler returned nil result")
		require.Len(t, result.Content, 1, "Expected exactly one content item in result")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in result")
		require.NotEmpty(t, textContent.Text, "Result text content should not be empty")

		// Unmarshal and check specific fields
		var versionInfo map[string]any // Using map[string]any for flexibility
		err = json.Unmarshal([]byte(textContent.Text), &versionInfo)
		require.NoError(t, err, "Failed to unmarshal version JSON")
		assert.Contains(t, versionInfo, "ApiVersion", "Version info should contain ApiVersion")
		assert.NotEmpty(t, versionInfo["ApiVersion"], "ApiVersion should not be empty")
		assert.Contains(t, versionInfo, "Version", "Version info should contain Version")
		assert.NotEmpty(t, versionInfo["Version"], "Version should not be empty")
	})

	// Subtest: Create Volume
	// Verifies that:
	// - A POST request to /volumes/create proxies correctly.
	// - A volume with the specified name is created.
	// - The handler response reflects the created volume details.
	t.Run("Create Volume", func(t *testing.T) {
		createBody := fmt.Sprintf(`{"Name": "%s"}`, testVolumeName)
		request := mcp.CreateMCPRequest(map[string]any{
			"environmentId": float64(testLocalEndpointID),
			"method":        "POST",
			"dockerAPIPath": "/volumes/create",
			"headers": []any{
				map[string]any{"key": "Content-Type", "value": "application/json"},
			},
			"body": createBody,
		})

		handler := env.MCPServer.HandleDockerProxy()
		result, err := handler(env.Ctx, request)

		require.NoError(t, err, "Create Volume handler execution failed")
		require.NotNil(t, result, "Create Volume handler returned nil result")
		require.Len(t, result.Content, 1, "Expected one content item for Create Volume")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content for Create Volume")
		require.NotEmpty(t, textContent.Text, "Create Volume response text should not be empty")

		var volumeInfo map[string]any
		err = json.Unmarshal([]byte(textContent.Text), &volumeInfo)
		require.NoError(t, err, "Failed to unmarshal Create Volume response")
		assert.Equal(t, testVolumeName, volumeInfo["Name"], "Volume name in response mismatch")
	})

	// Subtest: List Volumes with Filter
	// Verifies that:
	// - A GET request to /volumes with a name filter proxies correctly.
	// - The response contains only the volume created earlier.
	t.Run("List Volumes with Filter", func(t *testing.T) {
		filterJSON := fmt.Sprintf(`{"name":["%s"]}`, testVolumeName)
		request := mcp.CreateMCPRequest(map[string]any{
			"environmentId": float64(testLocalEndpointID),
			"method":        "GET",
			"dockerAPIPath": "/volumes",
			"queryParams": []any{
				map[string]any{"key": "filters", "value": filterJSON},
			},
		})

		handler := env.MCPServer.HandleDockerProxy()
		result, err := handler(env.Ctx, request)

		require.NoError(t, err, "List Volumes handler execution failed")
		require.NotNil(t, result, "List Volumes handler returned nil result")
		require.Len(t, result.Content, 1, "Expected one content item for List Volumes")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content for List Volumes")
		require.NotEmpty(t, textContent.Text, "List Volumes response text should not be empty")

		var listResponse map[string][]map[string]any
		err = json.Unmarshal([]byte(textContent.Text), &listResponse)
		require.NoError(t, err, "Failed to unmarshal List Volumes response")
		require.Contains(t, listResponse, "Volumes", "List response missing 'Volumes' key")
		require.Len(t, listResponse["Volumes"], 1, "Expected exactly one volume in the filtered list")
		assert.Equal(t, testVolumeName, listResponse["Volumes"][0]["Name"], "Filtered volume name mismatch")
	})

	// Subtest: Remove Volume
	// Verifies that:
	// - A DELETE request to /volumes/{name} proxies correctly.
	// - The volume created earlier is successfully removed.
	// - The handler response is empty (reflecting Docker's 204 No Content).
	t.Run("Remove Volume", func(t *testing.T) {
		request := mcp.CreateMCPRequest(map[string]any{
			"environmentId": float64(testLocalEndpointID),
			"method":        "DELETE",
			"dockerAPIPath": "/volumes/" + testVolumeName,
		})

		handler := env.MCPServer.HandleDockerProxy()
		result, err := handler(env.Ctx, request)

		require.NoError(t, err, "Remove Volume handler execution failed")
		require.NotNil(t, result, "Remove Volume handler returned nil result")
		require.Len(t, result.Content, 1, "Expected one content item for Remove Volume")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content for Remove Volume")
		assert.Empty(t, textContent.Text, "Remove Volume response text should be empty for 204 No Content")
	})
}
