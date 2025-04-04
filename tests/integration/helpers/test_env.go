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
	RawClient *client.PortainerClient
	MCPServer *mcp.PortainerMCPServer
}

// NewTestEnv creates a new test environment with Portainer container and clients
func NewTestEnv(t *testing.T) *TestEnv {
	ctx := context.Background()

	portainer, err := containers.NewPortainerContainer(ctx)
	require.NoError(t, err, "Failed to start Portainer container")

	host, port := portainer.GetHostAndPort()
	serverURL := fmt.Sprintf("%s:%s", host, port)

	rawCli := client.NewPortainerClient(
		serverURL,
		portainer.GetAPIToken(),
		client.WithSkipTLSVerify(true),
	)

	mcpServer, err := mcp.NewPortainerMCPServer(serverURL, portainer.GetAPIToken(), ToolsPath)
	require.NoError(t, err, "Failed to create MCP server")

	return &TestEnv{
		Ctx:       ctx,
		Portainer: portainer,
		RawClient: rawCli,
		MCPServer: mcpServer,
	}
}

// Cleanup terminates the Portainer container
func (e *TestEnv) Cleanup(t *testing.T) {
	if err := e.Portainer.Terminate(e.Ctx); err != nil {
		t.Logf("Failed to terminate container: %v", err)
	}
}
