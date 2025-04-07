package tooldef

import (
	_ "embed"
	"os"
)

//go:embed tools.yaml
var ToolsFile []byte

// CreateToolsFileIfNotExists creates the tools.yaml file if it doesn't exist
// It returns true if the file already exists, false if it was created or an error occurred
func CreateToolsFileIfNotExists(path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.WriteFile(path, ToolsFile, 0644)
		if err != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}
