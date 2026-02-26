package tooldef

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateToolsFileIfNotExists(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "tooldef-test")
	require.NoError(t, err, "Failed to create temporary directory")
	defer func() { _ = os.RemoveAll(tempDir) }()

	t.Run("File Does Not Exist", func(t *testing.T) {
		// Define a path for a non-existent file
		filePath := filepath.Join(tempDir, "new-tools.yaml")

		// Verify the file doesn't exist initially
		_, err := os.Stat(filePath)
		assert.True(t, os.IsNotExist(err), "File should not exist before test")

		exists, err := CreateToolsFileIfNotExists(filePath)
		require.NoError(t, err, "Function should not return an error")
		assert.False(t, exists, "Function should return false when creating a new file")

		// Verify the file was created
		_, err = os.Stat(filePath)
		assert.NoError(t, err, "File should exist after function call")

		// Verify the file has the embedded content
		content, err := os.ReadFile(filePath)
		require.NoError(t, err, "Should be able to read the created file")
		assert.Equal(t, ToolsFile, content, "File should contain the embedded tools content")
	})

	t.Run("File Already Exists", func(t *testing.T) {
		// Define a path for an existing file
		filePath := filepath.Join(tempDir, "existing-tools.yaml")

		// Create a custom file
		customContent := []byte("# Custom tools file content")
		err := os.WriteFile(filePath, customContent, 0644)
		require.NoError(t, err, "Failed to create test file")

		exists, err := CreateToolsFileIfNotExists(filePath)
		require.NoError(t, err, "Function should not return an error")
		assert.True(t, exists, "Function should return true when file already exists")

		// Verify the file content was not changed
		content, err := os.ReadFile(filePath)
		require.NoError(t, err, "Should be able to read the existing file")
		assert.Equal(t, customContent, content, "Function should not modify an existing file")
	})

	t.Run("Error During File Creation", func(t *testing.T) {
		// Create a path in a non-existent directory to force a file creation error
		nonExistentDir := filepath.Join(tempDir, "this-directory-does-not-exist", "neither-does-this-one")
		filePath := filepath.Join(nonExistentDir, "tools.yaml")

		exists, err := CreateToolsFileIfNotExists(filePath)
		assert.Error(t, err, "Function should return an error when file creation fails")
		assert.False(t, exists, "Function should return false when an error occurs")
	})
}
