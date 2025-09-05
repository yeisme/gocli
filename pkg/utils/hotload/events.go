package hotload

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/yeisme/gocli/pkg/configs"
	"github.com/yeisme/gocli/pkg/utils/gitignore"
)

// WatchContext 持有监视器的运行时状态.
// 包含根路径、fsnotify 监视器、配置、.gitignore 解析器、缓存、去抖动计时器等.
type WatchContext struct {
	rootPath string
	watcher  *fsnotify.Watcher
	config   configs.HotloadConfig
	gi       *gitignore.GitIgnore

	cache            map[string]fileState
	debounceDuration time.Duration

	// debounce runtime
	timer          *time.Timer
	changeDetected bool
}

// runEventLoop 处理 fsnotify 事件，应用过滤、状态跟踪和去抖动逻辑.
func runEventLoop(ctx *WatchContext, hook Func) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case event, ok := <-ctx.watcher.Events:
				if !ok {
					return
				}
				handleEvent(ctx, event)
				if shouldDebounce(ctx) {
					armOrResetDebounce(ctx, func() {
						onDebounceFire(ctx, hook)
					})
				}
			case err, ok := <-ctx.watcher.Errors:
				if !ok {
					return
				}
				logger.Error().Msgf("Watcher error: %s", err)
			}
		}
	}()
	<-done
	return nil
}

// handleEvent 决定事件是否有意义并更新缓存与标志位.
func handleEvent(ctx *WatchContext, event fsnotify.Event) {
	logEventWithThrottle(event.Op.String(), event.Name)

	// Ignore paths based on built-in, user patterns and .gitignore
	if isPathIgnored(ctx, event.Name) {
		return
	}

	var isRealChange bool
	switch {
	case event.Has(fsnotify.Create):
		isRealChange = onCreate(ctx, event.Name)
	case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
		isRealChange = onRemoveOrRename(ctx, event.Name)
	case event.Has(fsnotify.Write):
		isRealChange = onWrite(ctx, event.Name)
	}

	if isRealChange {
		ctx.changeDetected = true
	}
}

// isPathIgnored 将忽略逻辑集中处理，并按原因（例如 .git、过滤器、.gitignore）记录一次性日志.
func isPathIgnored(ctx *WatchContext, name string) bool {
	// Ignore .git paths early
	if stringsContainsGit(name) {
		logIgnoreWithThrottle(".git patterns", name)
		return true
	}
	// Filters and ignore patterns
	if shouldIgnoreFile(name, ctx.config.Filter, ctx.config.IgnorePatterns) {
		logIgnoreWithThrottle("filters/patterns", name)
		return true
	}
	if ctx.config.GitIgnore && ctx.gi != nil && len(ctx.gi.GetPatterns()) > 0 {
		if ctx.gi.IsIgnored(name) {
			logIgnoreWithThrottle(".gitignore", name)
			return true
		}
	}
	return false
}

// onCreate 处理文件和目录的创建事件.
// 对目录视情况自动添加到 watcher；对文件计算状态（必要时计算 hash）并更新缓存.
func onCreate(ctx *WatchContext, name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	if info.IsDir() {
		// auto-add directories
		if ctx.config.Recursive && !shouldIgnoreDirectory(name, ctx.config.IgnorePatterns) {
			if err := ctx.watcher.Add(name); err != nil {
				logger.Warn().Msgf("Failed to add new directory to watcher: %s", name)
			} else {
				logger.Debug().Msgf("Added new directory to watcher: %s", name)
			}
		}
		return false
	}
	var hash string
	if isSignificantFile(name) {
		hash = calculateFileHash(name, info.Size())
	}
	ctx.cache[name] = fileState{modTime: info.ModTime(), size: info.Size(), hash: hash}
	logger.Debug().Msgf("New file created: %s", name)
	return true
}

// onRemoveOrRename 处理删除或重命名事件.
// 如果文件之前被跟踪，则从缓存移除并返回 true 表示发生了实际变更.
func onRemoveOrRename(ctx *WatchContext, name string) bool {
	if _, wasTracked := ctx.cache[name]; wasTracked {
		delete(ctx.cache, name)
		return true
	}
	return false
}

// onWrite 处理写入/更新事件，并判断这是否为真实的文件内容或元数据变更.
// 优先使用内容 hash 比较（针对重要文件），否则使用大小和修改时间的容差判断.
func onWrite(ctx *WatchContext, name string) bool {
	oldState, wasTracked := ctx.cache[name]
	info, err := os.Stat(name)
	if err != nil {
		if wasTracked {
			delete(ctx.cache, name)
			logger.Debug().Msgf("File %s: deleted after write event", name)
			return true
		}
		return false
	}

	// Build new state (with hash for significant files)
	var newHash string
	significant := isSignificantFile(name)
	if significant {
		newHash = calculateFileHash(name, info.Size())
	}
	newState := fileState{modTime: info.ModTime(), size: info.Size(), hash: newHash}

	// If file wasn't tracked, treat as a new file
	if !wasTracked {
		ctx.cache[name] = newState
		logger.Debug().Msgf("File %s: new file detected", name)
		return true
	}

	// Prefer hash comparison for significant files when available
	if significant && oldState.hash != "" && newState.hash != "" {
		if oldState.hash == newState.hash {
			logger.Trace().Msgf("File %s: no content change (hash unchanged)", name)
			return false
		}
		if detectEditorSavePattern(name, newState.size) && newState.size == 0 {
			logger.Debug().Msgf("File %s: detected editor save truncation, waiting for content...", name)
			ctx.cache[name] = newState
			return false
		}
		ctx.cache[name] = newState
		logger.Debug().Msgf("File %s: content changed", name)
		return true
	}

	// Fallback: use size and modTime tolerance
	const timeTolerance = 100 * time.Millisecond
	sizeChanged := newState.size != oldState.size
	timeChanged := newState.modTime.Sub(oldState.modTime).Abs() > timeTolerance
	if sizeChanged || timeChanged {
		if detectEditorSavePattern(name, newState.size) && newState.size == 0 {
			logger.Debug().Msgf("File %s: detected editor save truncation, waiting for content...", name)
			ctx.cache[name] = newState
			return false
		}
		ctx.cache[name] = newState
		logger.Debug().Msgf("File %s: metadata changed", name)
		return true
	}

	logger.Trace().Msgf("File %s: no significant change detected", name)
	return false
}

func stringsContainsGit(p string) bool {
	return stringsContains(filepath.ToSlash(p), ".git/")
}

func stringsContains(s, sub string) bool {
	return stringContains(s, sub)
}
