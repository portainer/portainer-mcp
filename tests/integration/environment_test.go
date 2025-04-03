package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deviantony/portainer-mcp/internal/mcp"
	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/tests/integration/helpers"
	mcpmodels "github.com/mark3labs/mcp-go/mcp"
	"github.com/portainer/client-api-go/v2/client/utils"
	apimodels "github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Test data constants
	testEndpointName = "test-endpoint"
	testTag1Name     = "tag1"
	testTag2Name     = "tag2"
)

// prepareTestEnvironment prepares the test environment for the tests
// It enables Edge Compute settings and creates an Edge Docker endpoint
func prepareTestEnvironment(t *testing.T, env *helpers.TestEnv) {
	host, port := env.Portainer.GetHostAndPort()
	serverAddr := fmt.Sprintf("%s:%s", host, port)
	tunnelAddr := fmt.Sprintf("%s:8000", host)

	err := env.Client.UpdateSettings(true, serverAddr, tunnelAddr)
	require.NoError(t, err, "Failed to update settings")

	_, err = env.Client.CreateEdgeDockerEndpoint(testEndpointName)
	require.NoError(t, err, "Failed to create Edge Docker endpoint")
}

// TestEnvironmentManagement is an integration test suite that verifies the complete
// lifecycle of environment management in Portainer MCP. It tests the retrieval and
// configuration of environments, including tag management, user access controls,
// and team access policies.
func TestEnvironmentManagement(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup(t)

	// Prepare the test environment
	prepareTestEnvironment(t, env)

	var environment models.Environment

	// Subtest: Environment Retrieval
	// Verifies that:
	// - The environment is correctly retrieved from the system
	// - The environment has the expected default properties (type, status)
	// - No tags, user accesses, or team accesses are initially assigned
	// - Compares MCP handler output with direct client API call result
	t.Run("Environment Retrieval", func(t *testing.T) {
		handler := env.MCPServer.HandleGetEnvironments()
		result, err := handler(env.Ctx, mcp.CreateMCPRequest(nil))
		require.NoError(t, err, "Failed to get environments via MCP handler")

		assert.Len(t, result.Content, 1, "Expected exactly one environment from MCP handler")
		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		assert.True(t, ok, "Expected text content in MCP response")

		var environments []models.Environment
		err = json.Unmarshal([]byte(textContent.Text), &environments)
		require.NoError(t, err, "Failed to unmarshal environments from MCP response")
		require.Len(t, environments, 1, "Expected exactly one environment after unmarshalling")

		environment = environments[0]

		// Fetch the same endpoint directly via the client
		rawEndpoint, err := env.Client.GetEndpoint(int64(environment.ID))
		require.NoError(t, err, "Failed to get endpoint directly via client")

		// Convert the raw endpoint to the expected Environment model using the package's converter
		expectedEnvironment := models.ConvertEndpointToEnvironment(rawEndpoint)

		// Compare the Environment struct from MCP handler with the one converted from the direct client call
		assert.Equal(t, expectedEnvironment, environment, "Mismatch between MCP handler environment and converted client environment")
	})

	// Subtest: Tag Management
	// Verifies that:
	// - New tags can be created in the system
	// - Multiple tags can be assigned to an environment simultaneously
	// - The environment correctly reflects the assigned tag IDs
	// - The tags are properly persisted in the endpoint configuration
	t.Run("Tag Management", func(t *testing.T) {
		tagId1, err := env.Client.CreateTag(testTag1Name)
		require.NoError(t, err, "Failed to create first tag")
		tagId2, err := env.Client.CreateTag(testTag2Name)
		require.NoError(t, err, "Failed to create second tag")

		request := mcp.CreateMCPRequest(map[string]any{
			"id":     float64(environment.ID),
			"tagIds": []any{float64(tagId1), float64(tagId2)},
		})

		handler := env.MCPServer.HandleUpdateEnvironmentTags()
		_, err = handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update environment tags via MCP handler")

		// Verify by fetching endpoint directly via client
		endpoint, err := env.Client.GetEndpoint(int64(environment.ID))
		require.NoError(t, err, "Failed to get endpoint via client after tag update")
		assert.ElementsMatch(t, []int64{tagId1, tagId2}, endpoint.TagIds, "Tag IDs mismatch (Client check)") // Use ElementsMatch for unordered comparison
	})

	// Subtest: User Access Management
	// Verifies that:
	// - User access policies can be assigned to an environment
	// - Multiple users with different access levels can be configured
	// - Access levels are correctly mapped to appropriate role IDs
	// - The environment's user access policies are properly updated and persisted
	t.Run("User Access Management", func(t *testing.T) {
		request := mcp.CreateMCPRequest(map[string]any{
			"id": float64(environment.ID),
			"userAccesses": []any{
				map[string]any{"id": float64(1), "access": "environment_administrator"},
				map[string]any{"id": float64(2), "access": "standard_user"},
			},
		})

		handler := env.MCPServer.HandleUpdateEnvironmentUserAccesses()
		_, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update environment user accesses via MCP handler")

		// Verify by fetching endpoint directly via client
		endpoint, err := env.Client.GetEndpoint(int64(environment.ID))
		require.NoError(t, err, "Failed to get endpoint via client after user access update")

		expectedUserAccesses := utils.BuildAccessPolicies[apimodels.PortainerUserAccessPolicies](map[int64]string{
			1: "environment_administrator",
			2: "standard_user",
		})
		assert.Equal(t, expectedUserAccesses, endpoint.UserAccessPolicies, "User access policies mismatch (Client check)")
	})

	// Subtest: Team Access Management
	// Verifies that:
	// - Team access policies can be assigned to an environment
	// - Multiple teams with different access levels can be configured
	// - Access levels are correctly mapped to appropriate role IDs
	// - The environment's team access policies are properly updated and persisted
	t.Run("Team Access Management", func(t *testing.T) {
		request := mcp.CreateMCPRequest(map[string]any{
			"id": float64(environment.ID),
			"teamAccesses": []any{
				map[string]any{"id": float64(1), "access": "environment_administrator"},
				map[string]any{"id": float64(2), "access": "standard_user"},
			},
		})

		handler := env.MCPServer.HandleUpdateEnvironmentTeamAccesses()
		_, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update environment team accesses via MCP handler")

		// Verify by fetching endpoint directly via client
		endpoint, err := env.Client.GetEndpoint(int64(environment.ID))
		require.NoError(t, err, "Failed to get endpoint via client after team access update")

		expectedTeamAccesses := utils.BuildAccessPolicies[apimodels.PortainerTeamAccessPolicies](map[int64]string{
			1: "environment_administrator",
			2: "standard_user",
		})
		assert.Equal(t, expectedTeamAccesses, endpoint.TeamAccessPolicies, "Team access policies mismatch (Client check)")
	})
}
