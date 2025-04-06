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
	testStackName        = "test-mcp-stack"
	testStackFile        = "version: '3'\nservices:\n  web:\n    image: nginx:latest"
	testStackFileUpdated = "version: '3'\nservices:\n  web:\n    image: nginx:alpine"
	testEdgeGroupName    = "test-stack-group"
)

// prepareStackManagementTestEnvironment creates a test environment group needed for stack tests
func prepareStackManagementTestEnvironment(t *testing.T, env *helpers.TestEnv) int {
	// First, enable Edge features in Portainer
	host, port := env.Portainer.GetHostAndPort()
	serverAddr := fmt.Sprintf("%s:%s", host, port)
	tunnelAddr := fmt.Sprintf("%s:8000", host)

	err := env.RawClient.UpdateSettings(true, serverAddr, tunnelAddr)
	require.NoError(t, err, "Failed to update settings to enable Edge features")

	// Create a test environment group for the stack to be associated with
	testGroupID, err := env.RawClient.CreateEdgeGroup(testEdgeGroupName, []int64{})
	require.NoError(t, err, "Failed to create test environment group via raw client")

	return int(testGroupID)
}

// TestStackManagement is an integration test suite that verifies the complete
// lifecycle of stack management in Portainer MCP. It tests stack creation,
// retrieval, file content retrieval, and updates.
func TestStackManagement(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup(t)

	// Prepare the test environment
	testGroupID := prepareStackManagementTestEnvironment(t, env)

	var testStackID int

	// Subtest: Stack Creation
	// Verifies that:
	// - A new stack can be created via the MCP handler
	// - The handler response indicates success with an ID
	// - The created stack exists in Portainer when checked directly via Raw Client
	t.Run("Stack Creation", func(t *testing.T) {
		handler := env.MCPServer.HandleCreateStack()
		request := mcp.CreateMCPRequest(map[string]any{
			"name":                testStackName,
			"file":                testStackFile,
			"environmentGroupIds": []any{float64(testGroupID)},
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to create stack via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")

		// Check for success message and extract ID for later tests
		assert.Contains(t, textContent.Text, "Stack created successfully with ID:", "Success message prefix mismatch")

		// Verify by fetching stacks directly via client and finding the created stack by name
		stack, err := env.RawClient.GetEdgeStackByName(testStackName)
		require.NoError(t, err, "Failed to get stack directly via client after creation")
		assert.Equal(t, testStackName, stack.Name, "Stack name mismatch")

		// Extract stack ID for subsequent tests
		testStackID = int(stack.ID)
	})

	// Subtest: Stack Listing
	// Verifies that:
	// - The stack list can be retrieved via the MCP handler
	// - The list contains the expected stack
	// - The stack data matches the expected properties
	t.Run("Stack Listing", func(t *testing.T) {
		handler := env.MCPServer.HandleGetStacks()
		result, err := handler(env.Ctx, mcp.CreateMCPRequest(nil))
		require.NoError(t, err, "Failed to get stacks via MCP handler")

		assert.Len(t, result.Content, 1, "Expected exactly one content block in the result")
		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		assert.True(t, ok, "Expected text content in MCP response")

		var retrievedStacks []models.Stack
		err = json.Unmarshal([]byte(textContent.Text), &retrievedStacks)
		require.NoError(t, err, "Failed to unmarshal retrieved stacks")
		require.Len(t, retrievedStacks, 1, "Expected exactly one stack after unmarshalling")

		stack := retrievedStacks[0]
		assert.Equal(t, testStackName, stack.Name, "Stack name mismatch")

		// Fetch the same stack directly via the client
		rawStack, err := env.RawClient.GetEdgeStack(int64(testStackID))
		require.NoError(t, err, "Failed to get stack directly via client")

		// Convert the raw stack to the expected Stack model
		expectedStack := models.ConvertEdgeStackToStack(rawStack)
		assert.Equal(t, expectedStack, stack, "Stack mismatch between MCP handler and direct client call")
	})

	// Subtest: Get Stack File
	// Verifies that:
	// - The stack file can be retrieved via the MCP handler
	// - The file content matches the content used during creation
	t.Run("Get Stack File", func(t *testing.T) {
		handler := env.MCPServer.HandleGetStackFile()
		request := mcp.CreateMCPRequest(map[string]any{
			"id": float64(testStackID),
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to get stack file via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")

		// Compare with the original content
		assert.Equal(t, testStackFile, textContent.Text, "Stack file content mismatch")
	})

	// Subtest: Stack Update
	// Verifies that:
	// - A stack can be updated via the MCP handler
	// - The handler response indicates success
	// - The stack file is updated when checked directly via Raw Client
	t.Run("Stack Update", func(t *testing.T) {
		handler := env.MCPServer.HandleUpdateStack()
		request := mcp.CreateMCPRequest(map[string]any{
			"id":                  float64(testStackID),
			"file":                testStackFileUpdated,
			"environmentGroupIds": []any{float64(testGroupID)},
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to update stack via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")
		assert.Contains(t, textContent.Text, "Stack updated successfully", "Success message mismatch")

		// Verify by fetching stack file directly via raw client
		updatedFile, err := env.RawClient.GetEdgeStackFile(int64(testStackID))
		require.NoError(t, err, "Failed to get stack file via raw client after update")
		assert.Equal(t, testStackFileUpdated, updatedFile, "Stack file was not updated correctly")
	})
}
