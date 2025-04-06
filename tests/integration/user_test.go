package integration

import (
	"encoding/json"
	"testing"

	"github.com/deviantony/portainer-mcp/internal/mcp"
	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/tests/integration/helpers"
	mcpmodels "github.com/mark3labs/mcp-go/mcp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUsername     = "test-mcp-user"
	testUserPassword = "testpassword"
	userRoleStandard = 2 // Portainer API role ID for Standard User
)

// prepareUserManagementTestEnvironment creates a test user and returns its ID
func prepareUserManagementTestEnvironment(t *testing.T, env *helpers.TestEnv) int {
	testUserID, err := env.RawClient.CreateUser(testUsername, testUserPassword, userRoleStandard)
	require.NoError(t, err, "Failed to create test user via raw client")
	return int(testUserID)
}

// TestUserManagement is an integration test suite that verifies the complete
// lifecycle of user management in Portainer MCP. It tests user listing
// and role updates.
func TestUserManagement(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup(t)

	testUserID := prepareUserManagementTestEnvironment(t, env)

	// Subtest: User Listing
	// Verifies listing users (admin + test user) via MCP handler and compares with direct API call.
	t.Run("User Listing", func(t *testing.T) {
		handler := env.MCPServer.HandleGetUsers()
		result, err := handler(env.Ctx, mcp.CreateMCPRequest(nil))
		require.NoError(t, err, "Failed to get users via MCP handler")

		require.Len(t, result.Content, 1, "Expected exactly one content block in the result")
		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")

		var retrievedUsers []models.User
		err = json.Unmarshal([]byte(textContent.Text), &retrievedUsers)
		require.NoError(t, err, "Failed to unmarshal retrieved users")

		require.Equal(t, len(retrievedUsers), 2, "Expected 2 users (admin and test user)")

		rawUsers, err := env.RawClient.ListUsers()
		require.NoError(t, err, "Failed to get users directly via client for comparison")

		expectedConvertedUsers := make([]models.User, 0, len(rawUsers))
		for _, rawUser := range rawUsers {
			expectedConvertedUsers = append(expectedConvertedUsers, models.ConvertToUser(rawUser))
		}

		assert.ElementsMatch(t, expectedConvertedUsers, retrievedUsers, "Mismatch between MCP handler users and converted client users")
	})

	// Subtest: User Role Update
	// Verifies updating the test user's role from standard to admin via the MCP handler.
	t.Run("User Role Update", func(t *testing.T) {
		handler := env.MCPServer.HandleUpdateUserRole()

		newRole := models.UserRoleAdmin
		updateRequest := mcp.CreateMCPRequest(map[string]any{
			"id":   float64(testUserID),
			"role": newRole,
		})

		result, err := handler(env.Ctx, updateRequest)
		require.NoError(t, err, "Failed to update test user role to '%s' via MCP handler", newRole)

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response for role update")
		assert.Contains(t, textContent.Text, "User updated successfully", "Success message mismatch for role update")

		rawUpdatedUser, err := env.RawClient.GetUser(testUserID)
		require.NoError(t, err, "Failed to get test user directly via client after role update")

		convertedUpdatedUser := models.ConvertToUser(rawUpdatedUser)
		assert.Equal(t, newRole, convertedUpdatedUser.Role, "User role was not updated to '%s' after conversion check", newRole)
	})
}
