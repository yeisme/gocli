// Package hotload 提供高性能的文件变更监控工具，用于触发热重载.
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

	// saveDetection 跟踪可能的编辑器保存操作（先截断为 0 字节，然后写入实际内容）的时间点
	saveDetectionMap   = make(map[string]time.Time)
	saveDetectionMutex sync.Mutex
)

// Func 定义热重载钩子函数的类型.
type Func func()

// fileState 存储用于检测真实变更的文件关键信息（修改时间、大小和内容哈希）.
type fileState struct {
	modTime time.Time
	size    int64
	hash    string // 对于小文件为内容的 MD5 哈希，对于大文件为基于内容的简单校验标识
}

// stateCache 表示从文件路径到其上一次已知状态的映射.
type stateCache map[string]fileState

// calculateFileHash 为小文件（<1MB）计算 MD5 哈希以检测内容变更
func calculateFileHash(filePath string, size int64) string {
	// 仅为小文件计算哈希以避免性能问题
	const maxHashSize = 1024 * 1024 // 1MB
	if size > maxHashSize {
		// 对于大文件，使用大小作为简单校验标识
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

// isSignificantFile 检查文件是否足够重要以使用基于哈希的变更检测
func isSignificantFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	// 源代码文件和配置文件视为重要文件
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

	// 也检查那些没有扩展名但可能重要的文件名
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

// newWatcherWithState 对目录进行初始扫描以构建状态缓存.
// 它遍历目录树，对每个文件读取元数据并记录状态.
func newWatcherWithState(rootPath string, recursive bool) (stateCache, error) {
	cache := make(stateCache)
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过 .git 目录及其内容
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
		// 如果不递归，则跳过子目录.
		if !recursive && d.IsDir() && path != rootPath {
			return filepath.SkipDir
		}
		return nil
	}

	if err := filepath.WalkDir(rootPath, walkFunc); err != nil {
		return nil, fmt.Errorf("failed to build initial state cache for %s: %w", rootPath, err)
	}
	return cache, nil
}

// shouldIgnoreFile 检查文件是否应基于过滤器或忽略模式被忽略
func shouldIgnoreFile(filePath string, filters []string, ignorePatterns []string) bool {
	// Convert path separators to forward slashes for consistent pattern matching
	normalizedPath := filepath.ToSlash(filePath)
	fileName := filepath.Base(filePath)

	// 始终忽略 .git 目录及其内容
	if strings.Contains(normalizedPath, ".git/") || strings.HasPrefix(normalizedPath, ".git/") {
		return true
	}

	// 始终忽略常见的临时或系统文件
	commonIgnorePatterns := []string{
		"*.tmp", "*.swp", "*.log", "~*", ".DS_Store", "Thumbs.db",
		"*.lock", "*.pid", "*.temp",
	}

	allIgnorePatterns := append(ignorePatterns, commonIgnorePatterns...)

	// 先检查忽略模式
	for _, pattern := range allIgnorePatterns {
		// 移除前导 ./ 或 .\ 以保持一致性
		cleanPattern := strings.TrimPrefix(strings.TrimPrefix(pattern, "./"), ".\\")
		cleanPattern = filepath.ToSlash(cleanPattern)

		// 检查文件名是否匹配
		if matched, _ := filepath.Match(cleanPattern, fileName); matched {
			return true
		}

		// Check full path match for patterns with path separators
		if strings.Contains(cleanPattern, "/") {
			if matched, _ := filepath.Match(cleanPattern, normalizedPath); matched {
				return true
			}
			// 也检查路径是否以该模式开头（用于目录模式）
			if strings.HasSuffix(cleanPattern, "/") || strings.HasSuffix(cleanPattern, "*") {
				if strings.HasPrefix(normalizedPath, strings.TrimSuffix(cleanPattern, "*")) {
					return true
				}
			}
		}
	}

	// 如果未指定过滤器，则不忽略（监视所有文件）
	if len(filters) == 0 {
		return false
	}

	// 检查文件是否匹配任一过滤器
	for _, filter := range filters {
		cleanFilter := filepath.ToSlash(filter)
		if matched, _ := filepath.Match(cleanFilter, fileName); matched {
			return false // File matches filter, don't ignore
		}
		// 也检查文件扩展名模式
		if strings.HasPrefix(filter, "*.") {
			ext := filepath.Ext(fileName)
			if ext == filter[1:] { // Remove the * from *.go -> .go
				return false
			}
		}
	}

	// 文件未匹配任何过滤器，忽略它
	return true
}

// shouldIgnoreDirectory 检查目录是否应被忽略
func shouldIgnoreDirectory(dirPath string, ignorePatterns []string) bool {
	// 将路径分隔符规范为正斜杠以便匹配
	normalizedPath := filepath.ToSlash(dirPath)
	dirName := filepath.Base(dirPath)

	// 始终忽略 .git 目录
	if strings.Contains(normalizedPath, ".git/") || strings.HasSuffix(normalizedPath, ".git") || dirName == ".git" {
		return true
	}

	// 始终忽略常见的目录
	commonIgnoreDirs := []string{
		"node_modules", "vendor", ".vscode", ".idea", "dist", "build",
		"tmp", "temp", ".cache", ".next", ".nuxt",
	}

	for _, ignoreDir := range commonIgnoreDirs {
		if dirName == ignoreDir || strings.HasSuffix(normalizedPath, "/"+ignoreDir) {
			return true
		}
	}

	// 检查用户定义的忽略模式
	for _, pattern := range ignorePatterns {
		cleanPattern := strings.TrimPrefix(strings.TrimPrefix(pattern, "./"), ".\\")
		cleanPattern = filepath.ToSlash(cleanPattern)

		// 检查目录名是否匹配
		if matched, _ := filepath.Match(cleanPattern, dirName); matched {
			return true
		}

		// 检查包含路径分隔符的模式是否匹配完整路径
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

// detectEditorSavePattern 检测当前文件变更是否属于编辑器保存的模式
// 许多编辑器保存文件时会先将文件截断为 0 字节，然后再写入实际内容
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

// WatchWithConfig 根据配置监控目录并触发热重载回调
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

// initializeFileStateCache 初始化文件状态缓存
func initializeFileStateCache(rootPath string, recursive bool) (map[string]fileState, error) {
	cache, err := newWatcherWithState(rootPath, recursive)
	if err != nil {
		return nil, err
	}
	logger.Debug().Msgf("Initial state cache built for '%s' with %d files.", rootPath, len(cache))
	return cache, nil
}

// loadGitIgnore 加载目录下的 .gitignore（如果启用）
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

// addDirectoriesToWatcher 向 fsnotify 递归添加需要监视的目录（受配置与 .gitignore 约束）
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
