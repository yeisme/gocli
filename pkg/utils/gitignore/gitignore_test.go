package gitignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGitIgnoreLines(t *testing.T) {
	lines := []string{
		"# This is a comment",
		"",
		"*.log",
		"node_modules/",
		"/build",
		"temp*",
		"!important.log",
	}

	gi := ParseGitIgnoreLines(lines)
	patterns := gi.GetPatterns()

	expected := []string{"*.log", "node_modules/", "/build", "temp*", "!important.log"}
	if len(patterns) != len(expected) {
		t.Errorf("Expected %d patterns, got %d", len(expected), len(patterns))
	}

	for i, pattern := range patterns {
		if pattern != expected[i] {
			t.Errorf("Expected pattern %s, got %s", expected[i], pattern)
		}
	}
}

func TestIsIgnored(t *testing.T) {
	patterns := []string{
		"*.log",
		"node_modules/",
		"/build",
		"temp*",
		"*.tmp",
	}

	gi := ParseGitIgnoreLines(patterns)

	testCases := []struct {
		path     string
		expected bool
	}{
		{"test.log", true},
		{"app.js", false},
		{"node_modules", true},
		{"node_modules/package", true},
		{"src/node_modules", true},
		{"build", true},
		{"src/build", false}, // /build means only root build
		{"temp123", true},
		{"temporary", true},
		{"file.tmp", true},
		{"src/file.tmp", true},
	}

	for _, tc := range testCases {
		result := gi.IsIgnored(tc.path)
		if result != tc.expected {
			t.Errorf("IsIgnored(%s): expected %v, got %v", tc.path, tc.expected, result)
		}
	}
}

func TestLoadGitIgnoreFromFile(t *testing.T) {
	// Create a temporary .gitignore file
	tempDir := t.TempDir()
	gitignorePath := filepath.Join(tempDir, ".gitignore")

	content := `# Comment
*.log
node_modules/
/build
temp*
`

	err := os.WriteFile(gitignorePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .gitignore file: %v", err)
	}

	gi, err := LoadGitIgnore(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to load .gitignore: %v", err)
	}

	patterns := gi.GetPatterns()
	expected := []string{"*.log", "node_modules/", "/build", "temp*"}

	if len(patterns) != len(expected) {
		t.Errorf("Expected %d patterns, got %d", len(expected), len(patterns))
	}

	for i, pattern := range patterns {
		if pattern != expected[i] {
			t.Errorf("Expected pattern %s, got %s", expected[i], pattern)
		}
	}
}

func TestLoadGitIgnoreFromDir(t *testing.T) {
	// Create a temporary directory with .gitignore
	tempDir := t.TempDir()
	gitignorePath := filepath.Join(tempDir, ".gitignore")

	content := `*.log
node_modules/`

	err := os.WriteFile(gitignorePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .gitignore file: %v", err)
	}

	gi, err := LoadGitIgnoreFromDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to load .gitignore from dir: %v", err)
	}

	if !gi.IsIgnored("test.log") {
		t.Error("Expected test.log to be ignored")
	}

	if !gi.IsIgnored("node_modules") {
		t.Error("Expected node_modules to be ignored")
	}

	if gi.IsIgnored("test.js") {
		t.Error("Expected test.js to not be ignored")
	}
}

func TestFilterIgnoredPaths(t *testing.T) {
	patterns := []string{"*.log", "node_modules/", "temp*"}
	gi := ParseGitIgnoreLines(patterns)

	paths := []string{
		"src/app.js",
		"test.log",
		"node_modules/package.json",
		"temp_file.txt",
		"README.md",
		"build/output.js",
	}

	filtered := gi.FilterIgnoredPaths(paths)
	expected := []string{
		"src/app.js",
		"README.md",
		"build/output.js",
	}

	if len(filtered) != len(expected) {
		t.Errorf("Expected %d filtered paths, got %d", len(expected), len(filtered))
	}

	for i, path := range filtered {
		if path != expected[i] {
			t.Errorf("Expected filtered path %s, got %s", expected[i], path)
		}
	}
}

func TestNonExistentGitIgnore(t *testing.T) {
	// Test loading non-existent .gitignore file
	gi, err := LoadGitIgnore("/non/existent/path/.gitignore")
	if err != nil {
		t.Fatalf("Expected no error for non-existent .gitignore, got: %v", err)
	}

	if len(gi.GetPatterns()) != 0 {
		t.Error("Expected empty patterns for non-existent .gitignore")
	}

	// Should not ignore anything when no patterns are loaded
	if gi.IsIgnored("test.log") {
		t.Error("Expected no ignoring when no patterns are loaded")
	}
}
