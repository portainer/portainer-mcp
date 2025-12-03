package tooldef

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func ResolveToolsPath(flagValue string, defaultName string) string {
	s := strings.TrimSpace(flagValue)
	if s != "" {
		expanded := os.ExpandEnv(s)
		if runtime.GOOS == "windows" {
			re := regexp.MustCompile(`%([A-Za-z0-9_]+)%`)
			expanded = re.ReplaceAllStringFunc(expanded, func(m string) string {
				name := strings.Trim(m, "%")
				val := os.Getenv(name)
				if val == "" {
					return m
				}
				return val
			})
		}
		return expanded
	}
	exe, err := os.Executable()
	if err != nil {
		return defaultName
	}
	return filepath.Join(filepath.Dir(exe), defaultName)
}
