package integration

import (
	"context"
	"fmt"
	"testing"

	mcpmodels "github.com/mark3labs/mcp-go/mcp"
	"github.com/portainer/portainer-mcp/internal/mcp"
	"github.com/portainer/portainer-mcp/tests/integration/containers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	toolsPath        = "../../internal/tooldef/tools.yaml"
	unsupportedImage = "portainer/portainer-ee:2.29.1" // Older version than SupportedPortainerVersion
)

// TestServerInitialization verifies that the Portainer MCP server
// can be successfully initialized with a real Portainer instance.
func TestServerInitialization(t *testing.T) {
	// Start a Portainer container
	ctx := context.Background()

	portainer, err := containers.NewPortainerContainer(ctx)
	require.NoError(t, err, "Failed to start Portainer container")

	// Ensure container is terminated at the end of the test
	defer func() {
		if err := portainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get the host and port for the Portainer API
	host, port := portainer.GetHostAndPort()
	serverURL := fmt.Sprintf("%s:%s", host, port)
	apiToken := portainer.GetAPIToken()

	// Create the MCP server - this is the main test objective
	mcpServer, err := mcp.NewPortainerMCPServer(serverURL, apiToken, toolsPath)

	// Assert the server was created successfully
	require.NoError(t, err, "Failed to create MCP server")
	require.NotNil(t, mcpServer, "MCP server should not be nil")
}

// TestServerInitializationUnsupportedVersion verifies that the Portainer MCP server
// correctly rejects initialization with an unsupported Portainer version.
func TestServerInitializationUnsupportedVersion(t *testing.T) {
	// Start a Portainer container with unsupported version
	ctx := context.Background()

	portainer, err := containers.NewPortainerContainer(ctx, containers.WithImage(unsupportedImage))
	require.NoError(t, err, "Failed to start unsupported Portainer container")

	// Ensure container is terminated at the end of the test
	defer func() {
		if err := portainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get the host and port for the Portainer API
	host, port := portainer.GetHostAndPort()
	serverURL := fmt.Sprintf("%s:%s", host, port)
	apiToken := portainer.GetAPIToken()

	// Try to create the MCP server - should fail with version error
	mcpServer, err := mcp.NewPortainerMCPServer(serverURL, apiToken, toolsPath)

	// Assert the server creation failed with correct error
	assert.Error(t, err, "Server creation should fail with unsupported version")
	assert.Contains(t, err.Error(), "unsupported Portainer server version", "Error should indicate version mismatch")
	assert.Nil(t, mcpServer, "Server should be nil when version check fails")
}

// TestServerInitializationDisabledVersionCheck verifies that the Portainer MCP server
// can successfully connect to unsupported Portainer versions when version check is disabled.
func TestServerInitializationDisabledVersionCheck(t *testing.T) {
	// Start a Portainer container with unsupported version
	ctx := context.Background()

	portainer, err := containers.NewPortainerContainer(ctx, containers.WithImage(unsupportedImage))
	require.NoError(t, err, "Failed to start unsupported Portainer container")

	// Ensure container is terminated at the end of the test
	defer func() {
		if err := portainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get the host and port for the Portainer API
	host, port := portainer.GetHostAndPort()
	serverURL := fmt.Sprintf("%s:%s", host, port)
	apiToken := portainer.GetAPIToken()

	// Create the MCP server with disabled version check - should succeed despite unsupported version
	mcpServer, err := mcp.NewPortainerMCPServer(serverURL, apiToken, toolsPath, mcp.WithDisableVersionCheck(true))

	// Assert the server was created successfully
	require.NoError(t, err, "Failed to create MCP server with disabled version check")
	require.NotNil(t, mcpServer, "MCP server should not be nil when version check is disabled")

	// Verify basic functionality by testing settings retrieval
	handler := mcpServer.HandleGetSettings()
	request := mcp.CreateMCPRequest(nil) // GetSettings doesn't require parameters

	result, err := handler(ctx, request)
	require.NoError(t, err, "Failed to get settings via MCP handler with disabled version check")
	require.NotNil(t, result, "Settings result should not be nil")
	require.Len(t, result.Content, 1, "Expected exactly one content block in settings result")

	// Verify the response contains valid content
	textContent, ok := result.Content[0].(mcpmodels.TextContent)
	require.True(t, ok, "Expected text content in settings MCP response")
	assert.NotEmpty(t, textContent.Text, "Settings response should not be empty")
}
