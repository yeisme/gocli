package hotload

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/yeisme/gocli/pkg/configs"
	"github.com/yeisme/gocli/pkg/utils/gitignore"
)

// WatchContext carries runtime state for the watcher.
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

// runEventLoop processes fsnotify events and applies filtering, state tracking and debouncing.
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

// handleEvent determines if an event is meaningful and updates cache/flags.
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

// isPathIgnored centralizes ignore logic and logs the reason once.
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

// onCreate handles create events for files and directories.
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

// onRemoveOrRename handles removal or rename events.
func onRemoveOrRename(ctx *WatchContext, name string) bool {
	if _, wasTracked := ctx.cache[name]; wasTracked {
		delete(ctx.cache, name)
		return true
	}
	return false
}

// onWrite handles write/update events and determines if it's a real change.
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
