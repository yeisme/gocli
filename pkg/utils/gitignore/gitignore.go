// Package gitignore provides utilities for parsing and matching .gitignore patterns.
package gitignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// GitIgnore represents a collection of gitignore patterns
type GitIgnore struct {
	patterns []string
}

// LoadGitIgnore loads and parses a .gitignore file from the specified path
func LoadGitIgnore(gitignorePath string) (*GitIgnore, error) {
	file, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty GitIgnore if file doesn't exist
			return &GitIgnore{patterns: make([]string, 0)}, nil
		}
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			return
		}
	}()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &GitIgnore{patterns: patterns}, nil
}

// LoadGitIgnoreFromDir loads .gitignore file from the specified directory
func LoadGitIgnoreFromDir(dirPath string) (*GitIgnore, error) {
	gitignorePath := filepath.Join(dirPath, ".gitignore")
	return LoadGitIgnore(gitignorePath)
}

// ParseGitIgnoreLines parses gitignore patterns from a slice of strings
func ParseGitIgnoreLines(lines []string) *GitIgnore {
	var patterns []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return &GitIgnore{patterns: patterns}
}

// GetPatterns returns all loaded patterns
func (gi *GitIgnore) GetPatterns() []string {
	return gi.patterns
}

// IsIgnored checks if a file or directory path should be ignored
// This is a simple implementation that handles basic gitignore patterns
func (gi *GitIgnore) IsIgnored(path string) bool {
	// Normalize path separators to forward slashes
	normalizedPath := filepath.ToSlash(path)

	for _, pattern := range gi.patterns {
		if gi.matchPattern(normalizedPath, pattern) {
			return true
		}
	}
	return false
}

// matchPattern performs basic pattern matching for gitignore rules
// This is a simplified implementation that handles the most common cases
func (gi *GitIgnore) matchPattern(path, pattern string) bool {
	// Handle negation patterns (starting with !)
	if strings.HasPrefix(pattern, "!") {
		// For simplicity, we don't handle negation in this basic implementation
		return false
	}

	// Check if pattern starts with / (anchored to root)
	isRootAnchored := strings.HasPrefix(pattern, "/")
	if isRootAnchored {
		pattern = pattern[1:] // Remove leading slash
	}

	// Handle directory patterns (ending with /)
	if strings.HasSuffix(pattern, "/") {
		pattern = pattern[:len(pattern)-1]
		// For directory patterns, match directories
		if isRootAnchored {
			// Root anchored: only match at the beginning
			pathParts := strings.Split(path, "/")
			if len(pathParts) > 0 && gi.simpleMatch(pathParts[0], pattern) {
				return true
			}
		} else {
			// Not root anchored: match anywhere in the path
			pathParts := strings.Split(path, "/")
			for _, part := range pathParts {
				if gi.simpleMatch(part, pattern) {
					return true
				}
			}
		}
		return false
	}

	// Handle root anchored patterns
	if isRootAnchored {
		return gi.matchRootAnchoredPath(path, pattern)
	}

	// Handle simple wildcard matching (not root anchored)
	return gi.matchPathWithWildcard(path, pattern)
}

// matchRootAnchoredPath handles patterns that start with / (root anchored)
func (gi *GitIgnore) matchRootAnchoredPath(path, pattern string) bool {
	// For root anchored patterns, the path should start with the pattern
	pathParts := strings.Split(path, "/")
	patternParts := strings.Split(pattern, "/")

	// Pattern must match from the beginning of the path
	if len(pathParts) < len(patternParts) {
		return false
	}

	for i, patternPart := range patternParts {
		if !gi.simpleMatch(pathParts[i], patternPart) {
			return false
		}
	}

	return true
}

// simpleMatch performs simple string matching with * wildcard support
func (gi *GitIgnore) simpleMatch(text, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if !strings.Contains(pattern, "*") {
		return text == pattern
	}

	// Simple wildcard matching
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		prefix, suffix := parts[0], parts[1]
		return strings.HasPrefix(text, prefix) && strings.HasSuffix(text, suffix) && len(text) >= len(prefix)+len(suffix)
	}

	// For more complex patterns, use a simple approach
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(text, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(text, pattern[:len(pattern)-1])
	}

	return text == pattern
}

// matchPathWithWildcard checks if a path matches a pattern, considering directory structure
func (gi *GitIgnore) matchPathWithWildcard(path, pattern string) bool {
	// Split path and pattern into components
	pathParts := strings.Split(path, "/")
	patternParts := strings.Split(pattern, "/")

	// If pattern has no slashes, match against any part of the path
	if len(patternParts) == 1 {
		for _, part := range pathParts {
			if gi.simpleMatch(part, pattern) {
				return true
			}
		}
		return false
	}

	// For patterns with slashes, do more specific matching
	if len(pathParts) < len(patternParts) {
		return false
	}

	// Check if the pattern matches from any position in the path
	for i := 0; i <= len(pathParts)-len(patternParts); i++ {
		match := true
		for j, patternPart := range patternParts {
			if !gi.simpleMatch(pathParts[i+j], patternPart) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

// FilterIgnoredPaths filters out ignored paths from a list of paths
func (gi *GitIgnore) FilterIgnoredPaths(paths []string) []string {
	var result []string
	for _, path := range paths {
		if !gi.IsIgnored(path) {
			result = append(result, path)
		}
	}
	return result
}
