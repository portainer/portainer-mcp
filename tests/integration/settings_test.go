package integration

import (
	"encoding/json"
	"testing"

	"github.com/deviantony/portainer-mcp/internal/mcp"
	"github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/tests/integration/helpers"
	go_mcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSettingsManagement is an integration test suite that verifies the retrieval
// of Portainer settings via the MCP handler.
func TestSettingsManagement(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup(t)

	// Subtest: Settings Retrieval
	// Verifies that:
	// - Settings can be correctly retrieved from the system via the MCP handler.
	// - The retrieved settings match the expected values after preparation.
	t.Run("Settings Retrieval", func(t *testing.T) {
		handler := env.MCPServer.HandleGetSettings()
		result, err := handler(env.Ctx, mcp.CreateMCPRequest(nil))
		require.NoError(t, err, "Failed to get settings via MCP handler")

		assert.Len(t, result.Content, 1, "Expected exactly one content block in the result")
		textContent, ok := result.Content[0].(go_mcp.TextContent)
		assert.True(t, ok, "Expected text content in response")

		// Unmarshal the result from the MCP handler into the local models.PortainerSettings struct
		var retrievedSettings models.PortainerSettings
		err = json.Unmarshal([]byte(textContent.Text), &retrievedSettings)
		require.NoError(t, err, "Failed to unmarshal retrieved settings")

		// Fetch settings directly via client to compare
		rawSettings, err := env.RawClient.GetSettings()
		require.NoError(t, err, "Failed to get settings directly via client for comparison")

		// Convert the raw settings using the package's conversion function
		expectedConvertedSettings := models.ConvertSettingsToPortainerSettings(rawSettings)

		// Compare the Settings struct from MCP handler with the one converted from the direct client call
		assert.Equal(t, expectedConvertedSettings, retrievedSettings, "Mismatch between MCP handler settings and converted client settings")
	})
}
