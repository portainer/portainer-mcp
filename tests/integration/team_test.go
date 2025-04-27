package integration

import (
	"encoding/json"
	"testing"

	mcpmodels "github.com/mark3labs/mcp-go/mcp"
	"github.com/portainer/portainer-mcp/internal/mcp"
	"github.com/portainer/portainer-mcp/pkg/portainer/models"
	"github.com/portainer/portainer-mcp/tests/integration/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTeamName         = "test-mcp-team"
	testTeamNewName      = "test-mcp-team-updated"
	testUser1Name        = "test-team-user1"
	testUser2Name        = "test-team-user2"
	testTeamUserPassword = "testpassword"
	teamUserRoleStandard = 2 // Portainer API role ID for Standard User
)

// prepareTeamManagementTestEnvironment creates test users that can be assigned to teams
func prepareTeamManagementTestEnvironment(t *testing.T, env *helpers.TestEnv) (int, int) {
	testUser1ID, err := env.RawClient.CreateUser(testUser1Name, testTeamUserPassword, teamUserRoleStandard)
	require.NoError(t, err, "Failed to create first test user via raw client")

	testUser2ID, err := env.RawClient.CreateUser(testUser2Name, testTeamUserPassword, teamUserRoleStandard)
	require.NoError(t, err, "Failed to create second test user via raw client")

	return int(testUser1ID), int(testUser2ID)
}

// TestTeamManagement is an integration test suite that verifies the complete
// lifecycle of team management in Portainer MCP. It tests team creation,
// listing, name updates, and member management.
func TestTeamManagement(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup(t)

	// Prepare the test environment
	testUser1ID, testUser2ID := prepareTeamManagementTestEnvironment(t, env)

	var testTeamID int

	// Subtest: Team Creation
	// Verifies that:
	// - A new team can be created via the MCP handler.
	// - The handler response indicates success with an ID.
	// - The created team exists in Portainer when checked directly via the Raw Client.
	t.Run("Team Creation", func(t *testing.T) {
		handler := env.MCPServer.HandleCreateTeam()
		request := mcp.CreateMCPRequest(map[string]any{
			"name": testTeamName,
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to create team via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")

		// Check for success message and extract ID for later tests
		assert.Contains(t, textContent.Text, "Team created successfully with ID:", "Success message prefix mismatch")

		// Verify by fetching teams directly via client and finding the created team by name
		team, err := env.RawClient.GetTeamByName(testTeamName)
		require.NoError(t, err, "Failed to get team directly via client after creation")
		assert.Equal(t, testTeamName, team.Name, "Team name mismatch")

		// Extract team ID for subsequent tests
		testTeamID = int(team.ID)
	})

	// Subtest: Team Listing
	// Verifies that:
	// - The team list can be retrieved via the MCP handler
	// - The list contains the expected number of teams (one, the test team)
	// - The team has the correct name property
	// - The team data matches the team obtained directly via Raw Client when converted to the same model
	t.Run("Team Listing", func(t *testing.T) {
		handler := env.MCPServer.HandleGetTeams()
		result, err := handler(env.Ctx, mcp.CreateMCPRequest(nil))
		require.NoError(t, err, "Failed to get teams via MCP handler")

		assert.Len(t, result.Content, 1, "Expected exactly one content block in the result")
		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		assert.True(t, ok, "Expected text content in MCP response")

		var retrievedTeams []models.Team
		err = json.Unmarshal([]byte(textContent.Text), &retrievedTeams)
		require.NoError(t, err, "Failed to unmarshal retrieved teams")
		require.Len(t, retrievedTeams, 1, "Expected exactly one team after unmarshalling")

		team := retrievedTeams[0]
		assert.Equal(t, testTeamName, team.Name, "Team name mismatch")

		// Fetch the same team directly via the client
		rawTeam, err := env.RawClient.GetTeam(int64(testTeamID))
		require.NoError(t, err, "Failed to get team directly via client")

		// Convert the raw team to the expected Team model
		rawMemberships, err := env.RawClient.ListTeamMemberships()
		require.NoError(t, err, "Failed to get team memberships directly via client")
		expectedTeam := models.ConvertToTeam(rawTeam, rawMemberships)
		assert.Equal(t, expectedTeam, team, "Team mismatch between MCP handler and direct client call")
	})

	// Subtest: Team Name Update
	// Verifies that:
	// - A team's name can be updated via the MCP handler
	// - The handler response indicates success
	// - The team name is actually updated when checked directly via Raw Client
	t.Run("Team Name Update", func(t *testing.T) {
		handler := env.MCPServer.HandleUpdateTeamName()
		request := mcp.CreateMCPRequest(map[string]any{
			"id":   float64(testTeamID),
			"name": testTeamNewName,
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update team name via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response for team name update")
		assert.Contains(t, textContent.Text, "Team name updated successfully", "Success message mismatch for team name update")

		// Verify by fetching team directly via raw client
		rawTeam, err := env.RawClient.GetTeam(int64(testTeamID))
		require.NoError(t, err, "Failed to get team directly via client after name update")
		assert.Equal(t, testTeamNewName, rawTeam.Name, "Team name was not updated")
	})

	// Subtest: Team Members Update
	// Verifies that:
	// - Team members can be updated via the MCP handler
	// - The handler response indicates success
	// - The team memberships are correctly updated when checked directly via Raw Client
	// - Both test users are properly assigned to the team
	t.Run("Team Members Update", func(t *testing.T) {
		handler := env.MCPServer.HandleUpdateTeamMembers()
		request := mcp.CreateMCPRequest(map[string]any{
			"id":      float64(testTeamID),
			"userIds": []any{float64(testUser1ID), float64(testUser2ID)},
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update team members via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response for team members update")
		assert.Contains(t, textContent.Text, "Team members updated successfully", "Success message mismatch for team members update")

		// Verify by fetching team directly via raw client
		rawTeam, err := env.RawClient.GetTeam(int64(testTeamID))
		require.NoError(t, err, "Failed to get team directly via client after member update")
		rawMemberships, err := env.RawClient.ListTeamMemberships()
		require.NoError(t, err, "Failed to get team memberships directly via client")
		expectedTeam := models.ConvertToTeam(rawTeam, rawMemberships)
		assert.ElementsMatch(t, []int{testUser1ID, testUser2ID}, expectedTeam.MemberIDs, "Team memberships mismatch")
	})
}
