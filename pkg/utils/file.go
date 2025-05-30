package utils

import (
	"os"
	"path/filepath"
)

var (
	ext = []string{".yaml", ".yml"}
)

func UserConfigPath() string {
	// ~/.gocli/.gocli.yaml
	homeDir, _ := os.UserHomeDir()
	
	// First check if any config file already exists
	for _, e := range ext {
		configPath := filepath.Join(homeDir, ".gocli", ".gocli"+e)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}
	
	// If no config file exists, return the default path for .yaml
	return filepath.Join(homeDir, ".gocli", ".gocli.yaml")
}

// ProjectConfigPath returns the path to the project configuration file.
func ProjectConfigPath(p string) string {
	if p == "" {
		// .gocli.yaml in the current directory
		
		for _, e := range ext {
			if _, err := os.Stat(filepath.Join(".", ".gocli"+e)); err == nil {
				return filepath.Join(".", ".gocli"+e)
			}
		}
	} else {
		// custom filepath
		return p
	}
	return filepath.Join(".", ".gocli.yaml")
}
