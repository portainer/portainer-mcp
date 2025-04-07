package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deviantony/portainer-mcp/internal/mcp"
	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/tests/integration/helpers"
	mcpmodels "github.com/mark3labs/mcp-go/mcp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAccessGroupName      = "test-access-group"
	testAccessGroupNewName   = "test-access-group-updated"
	testTeamAccessGroupName  = "test-team-for-access-group"
	testUserAccessGroupName  = "test-user-for-access-group"
	testAccGroupPassword     = "testpassword"
	accGroupUserRoleStandard = 2 // Portainer API role ID for Standard User
	accGroupEndpointName     = "test-endpoint-for-access-group"
)

// prepareAccessGroupTestEnvironment creates test resources needed for access group tests
// including users, teams, and environments
func prepareAccessGroupTestEnvironment(t *testing.T, env *helpers.TestEnv) (int, int, int) {
	host, port := env.Portainer.GetHostAndPort()
	serverAddr := fmt.Sprintf("%s:%s", host, port)
	tunnelAddr := fmt.Sprintf("%s:8000", host)

	err := env.RawClient.UpdateSettings(true, serverAddr, tunnelAddr)
	require.NoError(t, err, "Failed to update settings")

	// Create a test user
	testUserID, err := env.RawClient.CreateUser(testUserAccessGroupName, testAccGroupPassword, accGroupUserRoleStandard)
	require.NoError(t, err, "Failed to create test user via raw client")

	// Create a test team
	testTeamID, err := env.RawClient.CreateTeam(testTeamAccessGroupName)
	require.NoError(t, err, "Failed to create test team via raw client")

	// Create a test environment
	testEnvID, err := env.RawClient.CreateEdgeDockerEndpoint(accGroupEndpointName)
	require.NoError(t, err, "Failed to create test environment via raw client")

	return int(testUserID), int(testTeamID), int(testEnvID)
}

// TestAccessGroupManagement is an integration test suite that verifies the complete
// lifecycle of access group management in Portainer MCP. It tests creation, listing,
// name updates, user accesses, team accesses, and environment management.
func TestAccessGroupManagement(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup(t)

	// Prepare the test environment
	testUserID, testTeamID, testEnvID := prepareAccessGroupTestEnvironment(t, env)

	var testAccessGroupID int

	// Subtest: Access Group Creation
	// Verifies that:
	// - A new access group can be created via the HandleCreateAccessGroup handler
	// - The handler response indicates success with an ID
	// - The created access group exists in Portainer when checked directly via Raw Client
	t.Run("Access Group Creation", func(t *testing.T) {
		handler := env.MCPServer.HandleCreateAccessGroup()
		request := mcp.CreateMCPRequest(map[string]any{
			"name":           testAccessGroupName,
			"environmentIds": []any{float64(testEnvID)},
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to create access group via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")

		// Check for success message and extract ID for later tests
		assert.Contains(t, textContent.Text, "Access group created successfully with ID:", "Success message prefix mismatch")

		// Verify by fetching access group directly via raw client
		rawAccessGroup, err := env.RawClient.GetEndpointGroupByName(testAccessGroupName)
		require.NoError(t, err, "Failed to get access group directly via raw client")
		assert.Equal(t, testAccessGroupName, rawAccessGroup.Name, "Access group name mismatch")

		// Extract group ID for subsequent tests
		testAccessGroupID = int(rawAccessGroup.ID)
	})

	// Subtest: Access Groups Listing
	// Verifies that:
	// - The access group list can be retrieved via the HandleGetAccessGroups handler
	// - The list contains the expected access group
	// - The access group has the correct name and properties
	t.Run("Access Groups Listing", func(t *testing.T) {
		handler := env.MCPServer.HandleGetAccessGroups()
		result, err := handler(env.Ctx, mcp.CreateMCPRequest(nil))
		require.NoError(t, err, "Failed to get access groups via MCP handler")

		assert.Len(t, result.Content, 1, "Expected exactly one content block in the result")
		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		assert.True(t, ok, "Expected text content in MCP response")

		var retrievedAccessGroups []models.AccessGroup
		err = json.Unmarshal([]byte(textContent.Text), &retrievedAccessGroups)
		require.NoError(t, err, "Failed to unmarshal retrieved access groups")
		require.Len(t, retrievedAccessGroups, 2, "Expected exactly two access groups after unmarshalling")

		accessGroup := retrievedAccessGroups[1]
		assert.Equal(t, testAccessGroupName, accessGroup.Name, "Access group name mismatch")

		// Fetch the same access group directly via the client
		rawAccessGroup, err := env.RawClient.GetEndpointGroup(int64(testAccessGroupID))
		require.NoError(t, err, "Failed to get access group directly via client")

		// Convert the raw access group to the expected AccessGroup model
		rawEndpoints, err := env.RawClient.ListEndpoints()
		require.NoError(t, err, "Failed to list endpoints")

		expectedAccessGroup := models.ConvertEndpointGroupToAccessGroup(rawAccessGroup, rawEndpoints)
		assert.Equal(t, expectedAccessGroup, accessGroup, "Access group mismatch between MCP handler and direct client call")
	})

	// Subtest: Access Group Name Update
	// Verifies that:
	// - An access group's name can be updated via the HandleUpdateAccessGroupName handler
	// - The handler response indicates success
	// - The access group name is actually updated when checked directly via Raw Client
	t.Run("Access Group Name Update", func(t *testing.T) {
		handler := env.MCPServer.HandleUpdateAccessGroupName()
		request := mcp.CreateMCPRequest(map[string]any{
			"id":   float64(testAccessGroupID),
			"name": testAccessGroupNewName,
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update access group name via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")
		assert.Contains(t, textContent.Text, "Access group name updated successfully", "Success message mismatch")

		// Verify by fetching access group directly via raw client
		updatedAccessGroup, err := env.RawClient.GetEndpointGroup(int64(testAccessGroupID))
		require.NoError(t, err, "Failed to get access group directly via client")
		assert.Equal(t, testAccessGroupNewName, updatedAccessGroup.Name, "Access group name was not updated")
	})

	// Subtest: Access Group User Accesses Update
	// Verifies that:
	// - User access policies can be updated via the HandleUpdateAccessGroupUserAccesses handler
	// - The handler response indicates success
	// - The access policies are correctly updated when checked directly via Raw Client
	t.Run("Access Group User Accesses Update", func(t *testing.T) {
		handler := env.MCPServer.HandleUpdateAccessGroupUserAccesses()
		request := mcp.CreateMCPRequest(map[string]any{
			"id": float64(testAccessGroupID),
			"userAccesses": []any{
				map[string]any{"id": float64(testUserID), "access": "environment_administrator"},
			},
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update access group user accesses via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")
		assert.Contains(t, textContent.Text, "Access group user accesses updated successfully", "Success message mismatch")

		// Verify by fetching access group directly via raw client and checking user accesses
		updatedAccessGroup, err := env.RawClient.GetEndpointGroup(int64(testAccessGroupID))
		require.NoError(t, err, "Failed to get access group directly via client")

		rawEndpoints, err := env.RawClient.ListEndpoints()
		require.NoError(t, err, "Failed to list endpoints")

		convertedAccessGroup := models.ConvertEndpointGroupToAccessGroup(updatedAccessGroup, rawEndpoints)
		userAccess, exists := convertedAccessGroup.UserAccesses[testUserID]
		assert.True(t, exists, "User access policy not found")
		assert.Equal(t, "environment_administrator", userAccess, "User access level mismatch")
	})

	// Subtest: Access Group Team Accesses Update
	// Verifies that:
	// - Team access policies can be updated via the HandleUpdateAccessGroupTeamAccesses handler
	// - The handler response indicates success
	// - The access policies are correctly updated when checked directly via Raw Client
	t.Run("Access Group Team Accesses Update", func(t *testing.T) {
		handler := env.MCPServer.HandleUpdateAccessGroupTeamAccesses()
		request := mcp.CreateMCPRequest(map[string]any{
			"id": float64(testAccessGroupID),
			"teamAccesses": []any{
				map[string]any{"id": float64(testTeamID), "access": "standard_user"},
			},
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update access group team accesses via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")
		assert.Contains(t, textContent.Text, "Access group team accesses updated successfully", "Success message mismatch")

		// Verify by fetching access group directly via raw client and checking team accesses
		updatedAccessGroup, err := env.RawClient.GetEndpointGroup(int64(testAccessGroupID))
		require.NoError(t, err, "Failed to get access group directly via client")

		rawEndpoints, err := env.RawClient.ListEndpoints()
		require.NoError(t, err, "Failed to list endpoints")

		convertedAccessGroup := models.ConvertEndpointGroupToAccessGroup(updatedAccessGroup, rawEndpoints)
		teamAccess, exists := convertedAccessGroup.TeamAccesses[testTeamID]
		assert.True(t, exists, "Team access policy not found")
		assert.Equal(t, "standard_user", teamAccess, "Team access level mismatch")
	})

	// Subtest: Remove Environment From Access Group
	// Verifies that:
	// - An environment can be removed from an access group via the HandleRemoveEnvironmentFromAccessGroup handler
	// - The handler response indicates success
	// - The environment is actually removed when checked directly via Raw Client
	t.Run("Remove Environment From Access Group", func(t *testing.T) {
		handler := env.MCPServer.HandleRemoveEnvironmentFromAccessGroup()
		request := mcp.CreateMCPRequest(map[string]any{
			"id":            float64(testAccessGroupID),
			"environmentId": float64(testEnvID),
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to remove environment from access group via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")
		assert.Contains(t, textContent.Text, "Environment removed from access group successfully", "Success message mismatch")

		// Verify by fetching access group directly via raw client and checking environments
		updatedAccessGroup, err := env.RawClient.GetEndpointGroup(int64(testAccessGroupID))
		require.NoError(t, err, "Failed to get access group directly via client")

		rawEndpoints, err := env.RawClient.ListEndpoints()
		require.NoError(t, err, "Failed to list endpoints")

		convertedAccessGroup := models.ConvertEndpointGroupToAccessGroup(updatedAccessGroup, rawEndpoints)
		assert.ElementsMatch(t, []int{}, convertedAccessGroup.EnvironmentIds, "Environment was not removed from access group")
	})

	// Subtest: Add Environment To Access Group
	// Verifies that:
	// - An environment can be added back to an access group via the HandleAddEnvironmentToAccessGroup handler
	// - The handler response indicates success
	// - The environment is actually added when checked directly via Raw Client
	// Note: This test is run after the remove test to verify both operations work correctly
	t.Run("Add Environment To Access Group", func(t *testing.T) {
		handler := env.MCPServer.HandleAddEnvironmentToAccessGroup()
		request := mcp.CreateMCPRequest(map[string]any{
			"id":            float64(testAccessGroupID),
			"environmentId": float64(testEnvID),
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to add environment to access group via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")
		assert.Contains(t, textContent.Text, "Environment added to access group successfully", "Success message mismatch")

		// Verify by fetching access group directly via raw client and checking environments
		updatedAccessGroup, err := env.RawClient.GetEndpointGroup(int64(testAccessGroupID))
		require.NoError(t, err, "Failed to get access group directly via client")

		rawEndpoints, err := env.RawClient.ListEndpoints()
		require.NoError(t, err, "Failed to list endpoints")

		convertedAccessGroup := models.ConvertEndpointGroupToAccessGroup(updatedAccessGroup, rawEndpoints)
		assert.ElementsMatch(t, []int{testEnvID}, convertedAccessGroup.EnvironmentIds, "Environment was not added to access group")
	})
}
