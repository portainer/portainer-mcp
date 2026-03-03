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
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			return true, nil
		}
		return false, err
	}
	defer f.Close()
	_, err = f.Write(ToolsFile)
	if err != nil {
		return false, err
	}
	return false, nil
}
