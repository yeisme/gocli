package utils

import (
	"os"
	"path/filepath"
)

func UserConfigPath() string {
	// ~/.gocli/.gocli.yaml
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".gocli", ".gocli.yaml")
}

// ProjectConfigPath returns the path to the project configuration file.
func ProjectConfigPath(p string) string {
	if p == "" {
		// .gocli.yaml in the current directory
		ext := []string{".yaml", ".yml"}
		for _, e := range ext {
			if _, err := os.Stat(filepath.Join(".", ".gocli"+e)); err == nil {
				return filepath.Join(".", ".gocli"+e)
			}
		}
	} else {
		// custom filepath
		return p
	}
	return ""
}
