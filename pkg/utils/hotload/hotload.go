// Package hotload provides high-performance utilities for monitoring file changes and triggering hot-reloading.
package hotload

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/yeisme/gocli/pkg/configs"
	"github.com/yeisme/gocli/pkg/utils/fsop"
	"github.com/yeisme/gocli/pkg/utils/gitignore"
	"github.com/yeisme/gocli/pkg/utils/log"
)

var (
	logger             = log.GetLogger()
	logEventCountMap   = make(map[string]int)
	logEventCountMutex sync.Mutex

	// saveDetection tracks potential file saves (size 0 -> actual size pattern)
	saveDetectionMap   = make(map[string]time.Time)
	saveDetectionMutex sync.Mutex
)

// Func defines the type for the hot-reloading hook function.
type Func func()

// fileState stores the essential metadata and content hash of a file to detect real changes.
type fileState struct {
	modTime time.Time
	size    int64
	hash    string // MD5 hash of file content for small files, or content-based checksum
}

// stateCache is a map from file path to its last known state.
type stateCache map[string]fileState

// calculateFileHash computes MD5 hash for small files (< 1MB) to detect content changes
func calculateFileHash(filePath string, size int64) string {
	// Only calculate hash for small files to avoid performance issues
	const maxHashSize = 1024 * 1024 // 1MB
	if size > maxHashSize {
		// For large files, use size + modtime as a simple checksum
		return fmt.Sprintf("large:%d", size)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error().Msgf("Failed to close file %s: %v", filePath, err)
			return
		}
	}()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

// isSignificantFile checks if a file is significant enough to warrant hash-based change detection
func isSignificantFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Source code files and configuration files are significant
	significantExtensions := []string{
		".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".java", ".c", ".cpp", ".h", ".hpp",
		".rs", ".php", ".rb", ".cs", ".swift", ".kt", ".scala", ".clj", ".elm",
		".yaml", ".yml", ".json", ".toml", ".xml", ".ini", ".conf", ".cfg",
		".md", ".rst", ".txt", ".sql", ".sh", ".bash", ".zsh", ".fish",
		".html", ".css", ".scss", ".sass", ".less", ".vue", ".svelte",
		".dockerfile", ".makefile", ".cmake", ".gradle", ".maven", ".npm",
	}

	if slices.Contains(significantExtensions, ext) {
		return true
	}

	// Also check for files without extensions that might be significant
	fileName := strings.ToLower(filepath.Base(filePath))
	significantFiles := []string{
		"dockerfile", "makefile", "rakefile", "gemfile", "pipfile",
		"package.json", "composer.json", "cargo.toml", "build.gradle",
	}

	return slices.Contains(significantFiles, fileName)
}

// logEventWithThrottle 智能地记录事件日志，避免冗余
func logEventWithThrottle(op, name string) {
	logEventCountMutex.Lock()
	defer logEventCountMutex.Unlock()

	key := fmt.Sprintf("%s:%s", op, name)
	count := logEventCountMap[key]
	logEventCountMap[key] = count + 1

	// 只记录前几次事件，然后每隔一定次数记录一次
	if count < 3 {
		logger.Debug().Msgf("EVENT! Op: %s, Name: %s", op, name)
	} else if count%10 == 0 {
		logger.Debug().Msgf("EVENT! Op: %s, Name: %s (occurred %d times)", op, name, count+1)
	}
}

// logIgnoreWithThrottle 智能地记录忽略日志，避免冗余
func logIgnoreWithThrottle(reason, filename string) {
	logEventCountMutex.Lock()
	defer logEventCountMutex.Unlock()

	key := fmt.Sprintf("ignore:%s:%s", reason, filename)
	count := logEventCountMap[key]
	logEventCountMap[key] = count + 1

	// 只记录第一次忽略，然后每隔一定次数记录一次
	if count == 0 {
		logger.Debug().Msgf("Ignoring file %s based on %s", filename, reason)
	} else if count%20 == 0 {
		logger.Debug().Msgf("Ignoring file %s based on %s (ignored %d times)", filename, reason, count+1)
	}
}

// newWatcherWithState performs an initial scan of a directory to build the initial state cache.
// It walks the directory tree, runs os.Stat() on each file, and stores the metadata.
func newWatcherWithState(rootPath string, recursive bool) (stateCache, error) {
	cache := make(stateCache)
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory and its contents
		if strings.Contains(filepath.ToSlash(path), ".git/") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}

			var hash string
			if isSignificantFile(path) {
				hash = calculateFileHash(path, info.Size())
			}

			cache[path] = fileState{
				modTime: info.ModTime(),
				size:    info.Size(),
				hash:    hash,
			}
		}
		// If not recursive, skip subdirectories.
		if !recursive && d.IsDir() && path != rootPath {
			return filepath.SkipDir
		}
		return nil
	}

	if err := filepath.WalkDir(rootPath, walkFunc); err != nil {
		return nil, fmt.Errorf("failed to build initial state cache for %s: %w", rootPath, err)
	}
	return cache, nil
} // shouldIgnoreFile checks if a file should be ignored based on the filters and ignore patterns
func shouldIgnoreFile(filePath string, filters []string, ignorePatterns []string) bool {
	// Convert path separators to forward slashes for consistent pattern matching
	normalizedPath := filepath.ToSlash(filePath)
	fileName := filepath.Base(filePath)

	// Always ignore .git directory and its contents
	if strings.Contains(normalizedPath, ".git/") || strings.HasPrefix(normalizedPath, ".git/") {
		return true
	}

	// Always ignore common temporary and system files
	commonIgnorePatterns := []string{
		"*.tmp", "*.swp", "*.log", "~*", ".DS_Store", "Thumbs.db",
		"*.lock", "*.pid", "*.temp",
	}

	allIgnorePatterns := append(ignorePatterns, commonIgnorePatterns...)

	// Check ignore patterns first
	for _, pattern := range allIgnorePatterns {
		// Remove leading ./ or .\ from pattern for consistency
		cleanPattern := strings.TrimPrefix(strings.TrimPrefix(pattern, "./"), ".\\")
		cleanPattern = filepath.ToSlash(cleanPattern)

		// Check filename match
		if matched, _ := filepath.Match(cleanPattern, fileName); matched {
			return true
		}

		// Check full path match for patterns with path separators
		if strings.Contains(cleanPattern, "/") {
			if matched, _ := filepath.Match(cleanPattern, normalizedPath); matched {
				return true
			}
			// Also check if the path starts with the pattern (for directory patterns)
			if strings.HasSuffix(cleanPattern, "/") || strings.HasSuffix(cleanPattern, "*") {
				if strings.HasPrefix(normalizedPath, strings.TrimSuffix(cleanPattern, "*")) {
					return true
				}
			}
		}
	}

	// If no filters specified, don't ignore (watch all files)
	if len(filters) == 0 {
		return false
	}

	// Check if file matches any filter
	for _, filter := range filters {
		cleanFilter := filepath.ToSlash(filter)
		if matched, _ := filepath.Match(cleanFilter, fileName); matched {
			return false // File matches filter, don't ignore
		}
		// Also check file extension patterns
		if strings.HasPrefix(filter, "*.") {
			ext := filepath.Ext(fileName)
			if ext == filter[1:] { // Remove the * from *.go -> .go
				return false
			}
		}
	}

	// File doesn't match any filter, ignore it
	return true
}

// shouldIgnoreDirectory checks if a directory should be ignored
func shouldIgnoreDirectory(dirPath string, ignorePatterns []string) bool {
	// Convert path separators to forward slashes for consistent pattern matching
	normalizedPath := filepath.ToSlash(dirPath)
	dirName := filepath.Base(dirPath)

	// Always ignore .git directory
	if strings.Contains(normalizedPath, ".git/") || strings.HasSuffix(normalizedPath, ".git") || dirName == ".git" {
		return true
	}

	// Always ignore common directories
	commonIgnoreDirs := []string{
		"node_modules", "vendor", ".vscode", ".idea", "dist", "build",
		"tmp", "temp", ".cache", ".next", ".nuxt",
	}

	for _, ignoreDir := range commonIgnoreDirs {
		if dirName == ignoreDir || strings.HasSuffix(normalizedPath, "/"+ignoreDir) {
			return true
		}
	}

	// Check user-defined ignore patterns
	for _, pattern := range ignorePatterns {
		cleanPattern := strings.TrimPrefix(strings.TrimPrefix(pattern, "./"), ".\\")
		cleanPattern = filepath.ToSlash(cleanPattern)

		// Check directory name match
		if matched, _ := filepath.Match(cleanPattern, dirName); matched {
			return true
		}

		// Check full path match for patterns with path separators
		if strings.Contains(cleanPattern, "/") {
			if matched, _ := filepath.Match(cleanPattern, normalizedPath); matched {
				return true
			}
			// Also check if the path starts with the pattern (for directory patterns)
			if strings.HasSuffix(cleanPattern, "/") || strings.HasSuffix(cleanPattern, "*") {
				if strings.HasPrefix(normalizedPath, strings.TrimSuffix(cleanPattern, "*")) {
					return true
				}
			}
		}
	}

	return false
}

// detectEditorSavePattern detects if the current file change is part of an editor save pattern
// Many editors save files by truncating to 0 bytes first, then writing the actual content
func detectEditorSavePattern(filePath string, newSize int64) bool {
	saveDetectionMutex.Lock()
	defer saveDetectionMutex.Unlock()

	now := time.Now()

	if newSize == 0 {
		// File truncated to 0 - mark as potential save start
		saveDetectionMap[filePath] = now
		return true // This is likely the start of a save operation
	}

	// Check if this is the completion of a save operation
	if saveStart, exists := saveDetectionMap[filePath]; exists {
		timeSinceTruncation := now.Sub(saveStart)
		delete(saveDetectionMap, filePath) // Clean up

		// If the file was truncated recently (within 1 second) and now has content,
		// this is likely the completion of an editor save
		if timeSinceTruncation <= time.Second && newSize > 0 {
			return true
		}
	}

	return false
}

// WatchWithConfig monitors directories with configuration-based behavior
func WatchWithConfig(config configs.HotloadConfig, hotloadHook Func) error {
	if !config.Enabled {
		logger.Warn().Msg("Hot reload is disabled in configuration")
		return nil
	}

	watchDir := config.Dir
	if watchDir == "" {
		var err error
		watchDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	logger.Info().Msgf("Starting configurable watcher on %s (recursive=%t)", watchDir, config.Recursive)
	logger.Debug().Msgf("Filter: %v, IgnorePatterns: %v, Debounce: %dms",
		config.Filter, config.IgnorePatterns, config.Debounce)

	return baseDirWatcherWithConfig(watchDir, config, hotloadHook)
}

// Extract initialization of file state cache
func initializeFileStateCache(rootPath string, recursive bool) (map[string]fileState, error) {
	cache, err := newWatcherWithState(rootPath, recursive)
	if err != nil {
		return nil, err
	}
	logger.Debug().Msgf("Initial state cache built for '%s' with %d files.", rootPath, len(cache))
	return cache, nil
}

// Extract loading of .gitignore
func loadGitIgnore(rootPath string, enabled bool) (*gitignore.GitIgnore, error) {
	if !enabled {
		logger.Info().Msg("GitIgnore filtering disabled by configuration")
		return &gitignore.GitIgnore{}, nil
	}

	gi, err := gitignore.LoadGitIgnoreFromDir(rootPath)
	if err != nil {
		logger.Warn().Msgf("Failed to load .gitignore from %s: %v", rootPath, err)
		return &gitignore.GitIgnore{}, nil
	}

	patterns := gi.GetPatterns()
	if len(patterns) > 0 {
		logger.Info().Msg("Loaded .gitignore patterns")
	}
	return gi, nil
}

// Extract recursive directory addition
func addDirectoriesToWatcher(watcher *fsnotify.Watcher, rootPath string, config configs.HotloadConfig, gi *gitignore.GitIgnore) error {
	var subdirs []string
	var err error

	if config.GitIgnore && len(gi.GetPatterns()) > 0 {
		subdirs, err = fsop.ListAllSubdirectoriesWithCustomGitIgnore(rootPath, gi)
	} else {
		subdirs, err = fsop.ListAllSubdirectories(rootPath)
	}
	if err != nil {
		return fmt.Errorf("failed to list subdirectories: %w", err)
	}

	var filteredDirs []string
	for _, dir := range subdirs {
		if !shouldIgnoreDirectory(dir, config.IgnorePatterns) {
			filteredDirs = append(filteredDirs, dir)
		}
	}

	pathsToWatch := append(filteredDirs, rootPath)
	logger.Debug().Msgf("Adding %d directories to watcher", len(pathsToWatch))
	for _, p := range pathsToWatch {
		if err := watcher.Add(p); err != nil {
			logger.Warn().Msgf("Failed to add path '%s' to watcher, skipping: %v", p, err)
		}
	}
	return nil
}
