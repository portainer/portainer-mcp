package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/deviantony/portainer-mcp/tests/integration/containers"
	"github.com/portainer/client-api-go/v2/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortainerStatus(t *testing.T) {
	// Create a new context
	ctx := context.Background()

	// Start a new Portainer container
	portainer, err := containers.NewPortainerContainer(ctx)
	require.NoError(t, err, "Failed to start Portainer container")
	defer func() {
		if err := portainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Create client
	host, port := portainer.GetHostAndPort()

	cli := client.NewPortainerClient(
		fmt.Sprintf("%s:%s", host, port),
		portainer.GetAPIToken(),
		client.WithSkipTLSVerify(true),
	)

	// Send validation request
	status, err := cli.GetSystemStatus()
	require.NoError(t, err, "Failed to get system status")

	fmt.Printf("System status: %+v\n", status)

	environments, err := cli.ListEndpoints()
	require.NoError(t, err, "Failed to list endpoints")

	assert.Equal(t, 1, len(environments), "Expected 1 endpoint")
}

// A list resource test would look like:
// 1. Create Portainer test container
// 2. Create a Portainer client
// 3. Send the request to the Portainer API to create a new resource
// 4. Assert the response
// 5. Create MCP server
// 6. Setup the MCP handler
// 7. Send the MCP request
// 8. Assert the response and validate the resource was created

// A resource creation test would look like:
// 1. Create Portainer test container
// 2. Create MCP server
// 3. Setup the MCP handler
// 4. Send the MCP request
// 5. Assert the response
// 6. Create a Portainer client
// 7. Send the request to the Portainer API
// 8. Assert the response
