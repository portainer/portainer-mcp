package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deviantony/portainer-mcp/internal/mcp"
	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/tests/integration/containers"
	"github.com/deviantony/portainer-mcp/tests/integration/helpers"
	mcpmodels "github.com/mark3labs/mcp-go/mcp"
	"github.com/portainer/client-api-go/v2/client"
	portainermodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	toolsPath = "../../tools.yaml"

	// Test data constants
	testEndpointName = "test-endpoint"
	testTag1Name     = "tag1"
	testTag2Name     = "tag2"
)

// testEnv holds the test environment configuration and clients
type testEnv struct {
	ctx       context.Context
	portainer *containers.PortainerContainer
	cli       *client.PortainerClient
	mcpServer *mcp.PortainerMCPServer
}

// setupTestEnv creates a new test environment with Portainer container and clients
func setupTestEnv(t *testing.T) *testEnv {
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

	mcpServer, err := mcp.NewPortainerMCPServer(serverURL, portainer.GetAPIToken(), toolsPath)
	require.NoError(t, err, "Failed to create MCP server")

	return &testEnv{
		ctx:       ctx,
		portainer: portainer,
		cli:       cli,
		mcpServer: mcpServer,
	}
}

// TestEnvironmentManagement is an integration test suite that verifies the complete
// lifecycle of environment management in Portainer MCP. It tests the creation and
// configuration of environments, including tag management, user access controls,
// and team access policies.
func TestEnvironmentManagement(t *testing.T) {
	env := setupTestEnv(t)
	defer func() {
		if err := env.portainer.Terminate(env.ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Initialize environment settings
	host, port := env.portainer.GetHostAndPort()
	serverURL := fmt.Sprintf("%s:%s", host, port)
	err := env.cli.UpdateSettings(true, serverURL, fmt.Sprintf("%s:8000", host))
	require.NoError(t, err, "Failed to update settings")

	_, err = env.cli.CreateEdgeDockerEndpoint(testEndpointName)
	require.NoError(t, err, "Failed to create local Docker endpoint")

	var environment models.Environment

	// Subtest: Environment Creation and Retrieval
	t.Run("Environment Creation and Retrieval", func(t *testing.T) {
		handler := env.mcpServer.HandleGetEnvironments()
		result, err := handler(env.ctx, helpers.CreateMCPRequest(nil))
		require.NoError(t, err, "Failed to get environments")

		assert.Len(t, result.Content, 1, "Expected exactly one environment")
		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		assert.True(t, ok, "Expected text content in response")

		var environments []models.Environment
		err = json.Unmarshal([]byte(textContent.Text), &environments)
		require.NoError(t, err, "Failed to unmarshal environments")

		environment = environments[0]
		assert.Equal(t, testEndpointName, environment.Name, "Environment name mismatch")
		assert.Equal(t, "docker-edge-agent", environment.Type, "Environment type mismatch")
		assert.Equal(t, "active", environment.Status, "Environment status mismatch")
		assert.Empty(t, environment.TagIds, "Expected no tags initially")
		assert.Empty(t, environment.UserAccesses, "Expected no user accesses initially")
		assert.Empty(t, environment.TeamAccesses, "Expected no team accesses initially")
	})

	// Subtest: Tag Management
	t.Run("Tag Management", func(t *testing.T) {
		tagId1, err := env.cli.CreateTag(testTag1Name)
		require.NoError(t, err, "Failed to create first tag")
		tagId2, err := env.cli.CreateTag(testTag2Name)
		require.NoError(t, err, "Failed to create second tag")

		request := helpers.CreateMCPRequest(map[string]any{
			"id":     float64(environment.ID),
			"tagIds": []any{float64(tagId1), float64(tagId2)},
		})

		handler := env.mcpServer.HandleUpdateEnvironmentTags()
		_, err = handler(env.ctx, request)
		require.NoError(t, err, "Failed to update environment tags")

		endpoint, err := env.cli.GetEndpoint(int64(environment.ID))
		require.NoError(t, err, "Failed to get endpoint")
		assert.Equal(t, []int64{tagId1, tagId2}, endpoint.TagIds, "Tag IDs mismatch")
	})

	// Subtest: User Access Management
	t.Run("User Access Management", func(t *testing.T) {
		request := helpers.CreateMCPRequest(map[string]any{
			"id": float64(environment.ID),
			"userAccesses": []any{
				map[string]any{"id": float64(1), "access": "environment_administrator"},
				map[string]any{"id": float64(2), "access": "standard_user"},
			},
		})

		handler := env.mcpServer.HandleUpdateEnvironmentUserAccesses()
		_, err = handler(env.ctx, request)
		require.NoError(t, err, "Failed to update environment user accesses")

		endpoint, err := env.cli.GetEndpoint(int64(environment.ID))
		require.NoError(t, err, "Failed to get endpoint")
		expectedUserAccesses := portainermodels.PortainerUserAccessPolicies{
			"1": portainermodels.PortainerAccessPolicy{RoleID: int64(1)}, // environment_administrator
			"2": portainermodels.PortainerAccessPolicy{RoleID: int64(3)}, // standard_user
		}
		assert.Equal(t, expectedUserAccesses, endpoint.UserAccessPolicies, "User access policies mismatch")
	})

	// Subtest: Team Access Management
	t.Run("Team Access Management", func(t *testing.T) {
		request := helpers.CreateMCPRequest(map[string]any{
			"id": float64(environment.ID),
			"teamAccesses": []any{
				map[string]any{"id": float64(1), "access": "environment_administrator"},
				map[string]any{"id": float64(2), "access": "standard_user"},
			},
		})

		handler := env.mcpServer.HandleUpdateEnvironmentTeamAccesses()
		_, err = handler(env.ctx, request)
		require.NoError(t, err, "Failed to update environment team accesses")

		endpoint, err := env.cli.GetEndpoint(int64(environment.ID))
		require.NoError(t, err, "Failed to get endpoint")
		expectedTeamAccesses := portainermodels.PortainerTeamAccessPolicies{
			"1": portainermodels.PortainerAccessPolicy{RoleID: int64(1)}, // environment_administrator
			"2": portainermodels.PortainerAccessPolicy{RoleID: int64(3)}, // standard_user
		}
		assert.Equal(t, expectedTeamAccesses, endpoint.TeamAccessPolicies, "Team access policies mismatch")
	})
}
