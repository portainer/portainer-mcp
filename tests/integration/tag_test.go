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
	testTagName1 = "test-tag-integration-1"
	testTagName2 = "test-tag-integration-2"
)

// TestTagManagement is an integration test suite that verifies the create
// and list operations for environment tags in Portainer MCP.
func TestTagManagement(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup(t)

	// Subtest: Tag Creation
	// Verifies that:
	// - A new tag can be created via the MCP handler.
	// - The handler response indicates success.
	// - The created tag exists in Portainer when checked directly via the Raw Client.
	t.Run("Tag Creation", func(t *testing.T) {
		handler := env.MCPServer.HandleCreateEnvironmentTag()
		request := mcp.CreateMCPRequest(map[string]any{
			"name": testTagName1,
		})

		result, err := handler(env.Ctx, request)
		require.NoError(t, err, "Failed to create tag via MCP handler")

		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")
		// Just check for the success prefix, no need to parse ID here
		assert.Contains(t, textContent.Text, "Environment tag created successfully with ID:", "Success message prefix mismatch")

		// Verify by fetching the tag directly via the client and finding the created tag by name
		tag, err := env.RawClient.GetTagByName(testTagName1)
		require.NoError(t, err, "Failed to get tag directly via client after creation")
		assert.Equal(t, testTagName1, tag.Name, "Tag name mismatch")
	})

	// Subtest: Tag Listing
	// Verifies that:
	// - Tags can be listed via the MCP handler.
	// - The list includes previously created tags.
	// - The data structure returned by the handler matches the expected local model.
	// - Compares MCP handler output with direct client API call result after conversion.
	t.Run("Tag Listing", func(t *testing.T) {
		// Create another tag directly for listing comparison
		_, err := env.RawClient.CreateTag(testTagName2)
		require.NoError(t, err, "Failed to create second tag directly")

		handler := env.MCPServer.HandleGetEnvironmentTags()
		result, err := handler(env.Ctx, mcp.CreateMCPRequest(nil))
		require.NoError(t, err, "Failed to get tags via MCP handler")

		require.Len(t, result.Content, 1, "Expected exactly one content block in the result")
		textContent, ok := result.Content[0].(mcpmodels.TextContent)
		require.True(t, ok, "Expected text content in MCP response")

		// Unmarshal the result from the MCP handler
		var retrievedTags []models.EnvironmentTag
		err = json.Unmarshal([]byte(textContent.Text), &retrievedTags)
		require.NoError(t, err, "Failed to unmarshal retrieved tags")

		// Fetch tags directly via client
		rawTags, err := env.RawClient.ListTags()
		require.NoError(t, err, "Failed to get tags directly via client for comparison")

		// Convert the raw tags to the expected EnvironmentTag model
		expectedConvertedTags := make([]models.EnvironmentTag, 0, len(rawTags))
		for _, rawTag := range rawTags {
			expectedConvertedTags = append(expectedConvertedTags, models.ConvertTagToEnvironmentTag(rawTag))
		}

		// Compare the tags from MCP handler with the ones converted from the direct client call
		// Use ElementsMatch as the order might not be guaranteed.
		assert.ElementsMatch(t, expectedConvertedTags, retrievedTags, "Mismatch between MCP handler tags and converted client tags")
	})
}
