package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deviantony/portainer-mcp/internal/mcp"
	pm_models "github.com/deviantony/portainer-mcp/pkg/portainer/models"
	"github.com/deviantony/portainer-mcp/tests/integration/helpers"
	go_mcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// prepareTestEnvironmentSettings prepares the test environment for the tests.
// It enables Edge Compute settings.
func prepareTestEnvironmentSettings(t *testing.T, env *helpers.TestEnv) {
	host, port := env.Portainer.GetHostAndPort()
	serverAddr := fmt.Sprintf("%s:%s", host, port)
	tunnelAddr := fmt.Sprintf("%s:8000", host)

	// Enable Edge Compute settings which is a prerequisite for some setting retrieval
	err := env.Client.UpdateSettings(true, serverAddr, tunnelAddr)
	require.NoError(t, err, "Failed to update settings for test preparation")
}

// TestSettingsManagement is an integration test suite that verifies the retrieval
// of Portainer settings via the MCP handler.
func TestSettingsManagement(t *testing.T) {
	env := helpers.NewTestEnv(t)
	defer env.Cleanup(t)

	// Prepare the test environment
	prepareTestEnvironmentSettings(t, env)

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
		var retrievedSettings pm_models.PortainerSettings
		err = json.Unmarshal([]byte(textContent.Text), &retrievedSettings)
		require.NoError(t, err, "Failed to unmarshal retrieved settings")

		// Fetch settings directly via client to compare
		// Note: env.Client.GetSettings() returns the raw client-api-go settings struct
		rawSettings, err := env.Client.GetSettings()
		require.NoError(t, err, "Failed to get settings directly via client for comparison")

		// Convert the raw settings using the package's conversion function
		expectedConvertedSettings := pm_models.ConvertSettingsToPortainerSettings(rawSettings)

		// Compare the Settings struct from MCP handler with the one converted from the direct client call
		assert.Equal(t, expectedConvertedSettings, retrievedSettings, "Mismatch between MCP handler settings and converted client settings")
	})
}
