package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	mcpmodels "github.com/mark3labs/mcp-go/mcp"
	"github.com/portainer/portainer-mcp/internal/mcp"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/portainer/portainer-mcp/tests/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testGroupName        = "test-mcp-group"
	testGroupUpdatedName = "test-mcp-group-updated"
	testGroupTagName1    = "test-group-tag1"
	testGroupTagName2    = "test-group-tag2"
	testEnvName          = "test-group-env"
)

// prepareEnvironmentGroupTestEnvironment prepares the test environment for environment group tests
func prepareEnvironmentGroupTestEnvironment(t *testing.T, env *helpers.TestEnv) (int, int) {
	// Enable Edge features in Portainer
	host, port := env.Portainer.GetHostAndPort()
	serverAddr := fmt.Sprintf("%s:%s", host, port)
	tunnelAddr := fmt.Sprintf("%s:8000", host)

	err := env.RawClient.UpdateSettings(true, serverAddr, tunnelAddr)
	require.NoError(t, err, "Failed to update settings to enable Edge features")

	// Create a test environment for association with groups
	envID, err := env.RawClient.CreateEdgeDockerEndpoint(testEnvName)
	require.NoError(t, err, "Failed to create test environment")

	// Create test tag
	tagID, err := env.RawClient.CreateTag(testGroupTagName1)
	require.NoError(t, err, "Failed to create test tag")

	return int(envID), int(tagID)
}

// TestEnvironmentGroupManagement is an integration test suite that verifies the complete
// lifecycle of environment group management in Portainer MCP. It tests group creation,
// listing, name updates, environment association and tag association.
func TestEnvironmentGroupManagement(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup(t)

	// Prepare the test environment
	testEnvID, testTagID := prepareEnvironmentGroupTestEnvironment(t, env)

	var testGroupID int

	// Subtest: Environment Group Creation
	// Verifies that:
	// - A new environment group can be created via the MCP handler
	// - The handler response indicates success with an ID
	// - The created group exists in Portainer when checked directly via Raw Client
	t.Run("Environment Group Creation", func(t *testing.T) {
		handler := env.MCPServer.HandleCreateEnvironmentGroup()
		request := mcp.CreateMCPRequest(map[string]any{
			"name":           testGroupName,
			"environmentIds": []any{float64(testEnvID)},
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to create environment group via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")

		// Check for success message
		assert.Contains(t, textContent.Text, "Environment group created successfully with ID:", "Success message prefix mismatch")

		// Verify by fetching group directly via client and finding the created group by name
		group, err := env.RawClient.GetEdgeGroupByName(testGroupName)
		require.NoError(t, err, "Failed to get environment group directly via client")
		assert.Equal(t, testGroupName, group.Name, "Group name mismatch")

		// Extract group ID for subsequent tests
		testGroupID = int(group.ID)
	})

	// Subtest: Environment Group Listing
	// Verifies that:
	// - The group list can be retrieved via the MCP handler
	// - The list contains the expected group
	// - The group data matches the expected properties
	t.Run("Environment Group Listing", func(t *testing.T) {
		handler := env.MCPServer.HandleGetEnvironmentGroups()
		result, err := handler(env.Ctx, mcp.CreateMCPRequest(nil))
		require.NoError(t, err, "Failed to get environment groups via MCP handler")

		assert.Len(t, result.Content, 1, "Expected exactly one content block in the result")
		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		assert.True(t, ok, "Expected text content in MCP response")

		var retrievedGroups []models.Group
		err = json.Unmarshal([]byte(textContent.Text), &retrievedGroups)
		require.NoError(t, err, "Failed to unmarshal retrieved groups")
		require.Len(t, retrievedGroups, 1, "Expected exactly one group after unmarshalling")

		group := retrievedGroups[0]
		assert.Equal(t, testGroupName, group.Name, "Group name mismatch")

		// Fetch the same group directly via the client
		rawGroup, err := env.RawClient.GetEdgeGroup(int64(testGroupID))
		require.NoError(t, err, "Failed to get environment group directly via client")

		// Convert the raw group to the expected Group model
		expectedGroup := models.ConvertEdgeGroupToGroup(rawGroup)
		assert.Equal(t, expectedGroup, group, "Group mismatch between MCP handler and direct client call")
	})

	// Subtest: Environment Group Name Update
	// Verifies that:
	// - The group name can be updated via the MCP handler
	// - The handler response indicates success
	// - The name is correctly updated when checked directly via Raw Client
	t.Run("Environment Group Name Update", func(t *testing.T) {
		handler := env.MCPServer.HandleUpdateEnvironmentGroupName()
		request := mcp.CreateMCPRequest(map[string]any{
			"id":   float64(testGroupID),
			"name": testGroupUpdatedName,
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update environment group name via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")
		assert.Contains(t, textContent.Text, "Environment group name updated successfully", "Success message mismatch")

		// Verify by fetching group directly via raw client
		updatedGroup, err := env.RawClient.GetEdgeGroup(int64(testGroupID))
		require.NoError(t, err, "Failed to get environment group directly via client")
		assert.Equal(t, testGroupUpdatedName, updatedGroup.Name, "Group name was not updated")
	})

	// Subtest: Environment Group Tag Update
	// Verifies that:
	// - Tags can be associated with a group via the MCP handler
	// - The handler response indicates success
	// - The tags are correctly associated when checked directly via Raw Client
	t.Run("Environment Group Tag Update", func(t *testing.T) {
		// Create a second tag
		tagID2, err := env.RawClient.CreateTag(testGroupTagName2)
		require.NoError(t, err, "Failed to create second test tag")

		handler := env.MCPServer.HandleUpdateEnvironmentGroupTags()
		request := mcp.CreateMCPRequest(map[string]any{
			"id":     float64(testGroupID),
			"name":   testGroupUpdatedName,
			"tagIds": []any{float64(testTagID), float64(tagID2)},
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update environment group tags via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")
		assert.Contains(t, textContent.Text, "Environment group tags updated successfully", "Success message mismatch")

		// Verify by fetching group directly via raw client
		updatedGroup, err := env.RawClient.GetEdgeGroup(int64(testGroupID))
		require.NoError(t, err, "Failed to get environment group directly via client")
		assert.ElementsMatch(t, []int64{int64(testTagID), int64(tagID2)}, updatedGroup.TagIds, "Tag IDs mismatch")
	})

	// Subtest: Environment Group Environments Update
	// Verifies that:
	// - Environment associations can be updated via the MCP handler
	// - The handler response indicates success
	// - The environment associations are correctly updated when checked directly via Raw Client
	t.Run("Environment Group Environments Update", func(t *testing.T) {
		// Create a second environment
		env2Name := "test-env-2"
		env2ID, err := env.RawClient.CreateEdgeDockerEndpoint(env2Name)
		require.NoError(t, err, "Failed to create second test environment")

		handler := env.MCPServer.HandleUpdateEnvironmentGroupEnvironments()
		request := mcp.CreateMCPRequest(map[string]any{
			"id":             float64(testGroupID),
			"name":           testGroupUpdatedName,
			"environmentIds": []any{float64(testEnvID), float64(env2ID)},
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update environment group environments via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")
		assert.Contains(t, textContent.Text, "Environment group environments updated successfully", "Success message mismatch")

		// Verify by fetching group directly via raw client
		updatedGroup, err := env.RawClient.GetEdgeGroup(int64(testGroupID))
		require.NoError(t, err, "Failed to get environment group directly via client")
		assert.ElementsMatch(t, []int64{int64(testEnvID), int64(env2ID)}, updatedGroup.Endpoints, "Environment IDs mismatch")
	})
}
