package parse

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestParseConfig demonstrates configuration file parsing
func TestParseConfig(t *testing.T) {
	fmt.Println("\n=== Configuration File Parsing Demo ===")

	// Sample configuration YAML
	configYAML := `
name: gocli
version: "1"

project:
  name: demo-project
  version: "1.0.0"
  description: "Demo project"
  author: "yeisme"
  license: "MIT"

build:
  - name: default
    description: "Build project"
    cmd:
      - go build
    silent: true

test:
  - name: default
    description: "Run tests"
    cmd:
      - go test ./...
    silent: false
`

	config, err := ParseConfig([]byte(configYAML))
	if err != nil {
		t.Fatalf("Failed to parse configuration: %v", err)
	}

	fmt.Printf("Project information:\n")
	fmt.Printf("  Name: %s\n", config.Project.Name)
	fmt.Printf("  Version: %s\n", config.Project.Version)
	fmt.Printf("  Description: %s\n", config.Project.Description)
	fmt.Printf("  Author: %s\n", config.Project.Author)

	fmt.Printf("\nBuild configuration:\n")
	for _, build := range config.Build {
		fmt.Printf("  - %s: %s\n", build.Name, build.Description)
		for _, cmd := range build.Cmd {
			fmt.Printf("    Command: %s\n", cmd)
		}
	}
}

// TestParseConfigFromFile demonstrates parsing example configuration file
func TestParseConfigFromFile(t *testing.T) {
	fmt.Println("\n=== Example Configuration File Parsing Demo ===")

	// Get project root directory
	currentDir, _ := os.Getwd()
	projectRoot := filepath.Join(currentDir, "..", "..")
	configPath := filepath.Join(projectRoot, ".gocli.example.yaml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skipf("Example configuration file does not exist: %s", configPath)
		return
	}

	config, err := ParseConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to parse example configuration file: %v", err)
	} else {
		t.Logf("Successfully parsed example configuration file: %s", configPath)
	}

	// Set as project configuration
	SetProjectConfig(config)

	fmt.Printf("Project configuration loaded:\n")
	fmt.Printf("  Project name: %s\n", config.Project.Name)
	fmt.Printf("  Version: %s\n", config.Project.Version)
	fmt.Printf("  Go version: %s\n", config.Project.GoVersion)
	fmt.Printf("  Repository: %s\n", config.Project.Repository)

	fmt.Printf("\nAvailable commands:\n")

	// Show build commands
	if len(config.Build) > 0 {
		fmt.Printf("  Build commands:\n")
		for _, build := range config.Build {
			fmt.Printf("    - %s: %s\n", build.Name, build.Description)
		}
	}

	// Show test commands
	if len(config.Test) > 0 {
		fmt.Printf("  Test commands:\n")
		for _, test := range config.Test {
			fmt.Printf("    - %s: %s\n", test.Name, test.Description)
		}
	}

	// Show tools configuration
	if len(config.Tools.Go) > 0 {
		fmt.Printf("  Go tools:\n")
		for _, tool := range config.Tools.Go {
			fmt.Printf("    - %s: %s\n", tool.Name, tool.URL)
		}
	}

	// Verify configuration can be retrieved correctly
	retrievedConfig := GetProjectConfig()
	if retrievedConfig == nil {
		t.Error("Failed to get project configuration")
	} else {
		fmt.Printf("\n✓ Configuration successfully stored and retrieved\n")
	}
}
