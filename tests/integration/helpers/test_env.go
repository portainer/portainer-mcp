package helpers

import (
	"context"
	"fmt"
	"testing"

	"github.com/deviantony/portainer-mcp/internal/mcp"
	"github.com/deviantony/portainer-mcp/tests/integration/containers"
	"github.com/portainer/client-api-go/v2/client"
	"github.com/stretchr/testify/require"
)

const (
	ToolsPath = "../../tools.yaml"
)

// TestEnv holds the test environment configuration and clients
type TestEnv struct {
	Ctx       context.Context
	Portainer *containers.PortainerContainer
	Client    *client.PortainerClient
	MCPServer *mcp.PortainerMCPServer
}

// NewTestEnv creates a new test environment with Portainer container and clients
func NewTestEnv(t *testing.T) *TestEnv {
	ctx := context.Background()

	portainer, err := containers.NewPortainerContainer(ctx)
	require.NoError(t, err, "Failed to start Portainer container")

	host, port := portainer.GetHostAndPort()
	serverURL := fmt.Sprintf("%s:%s", host, port)

	cli := client.NewPortainerClient(
		serverURL,
		portainer.GetAPIToken(),
		client.WithSkipTLSVerify(true),
	)

	mcpServer, err := mcp.NewPortainerMCPServer(serverURL, portainer.GetAPIToken(), ToolsPath)
	require.NoError(t, err, "Failed to create MCP server")

	return &TestEnv{
		Ctx:       ctx,
		Portainer: portainer,
		Client:    cli,
		MCPServer: mcpServer,
	}
}

// Cleanup terminates the Portainer container
func (e *TestEnv) Cleanup(t *testing.T) {
	if err := e.Portainer.Terminate(e.Ctx); err != nil {
		t.Logf("Failed to terminate container: %v", err)
	}
}

// GetServerURL returns the server URL for the test environment
func (e *TestEnv) GetServerURL() string {
	host, port := e.Portainer.GetHostAndPort()
	return fmt.Sprintf("%s:%s", host, port)
}

// InitializeEnvironment sets up the initial environment settings
func (e *TestEnv) InitializeEnvironment(t *testing.T) {
	host, port := e.Portainer.GetHostAndPort()
	serverURL := fmt.Sprintf("%s:%s", host, port)
	err := e.Client.UpdateSettings(true, serverURL, fmt.Sprintf("%s:8000", host))
	require.NoError(t, err, "Failed to update settings")
}
