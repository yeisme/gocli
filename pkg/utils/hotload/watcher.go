package hotload

import (
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/yeisme/gocli/pkg/configs"
)

// baseDirWatcherWithConfig is a slim orchestrator that wires watcher + cache + filters, then starts the event loop.
func baseDirWatcherWithConfig(rootPath string, config configs.HotloadConfig, hook Func) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer func() {
		if cerr := watcher.Close(); cerr != nil {
			logger.Error().Msgf("Failed to close watcher: %v", cerr)
		}
	}()

	// Initial file state cache
	cache, err := initializeFileStateCache(rootPath, config.Recursive)
	if err != nil {
		return err
	}

	// Load .gitignore if enabled
	gi, err := loadGitIgnore(rootPath, config.GitIgnore)
	if err != nil {
		return err
	}

	// Register directories
	if config.Recursive {
		if err := addDirectoriesToWatcher(watcher, rootPath, config, gi); err != nil {
			return err
		}
	} else {
		if err := watcher.Add(rootPath); err != nil {
			return fmt.Errorf("failed to add root path '%s' to watcher: %w", rootPath, err)
		}
	}

	// Debounce duration
	debounceDuration := time.Duration(config.Debounce) * time.Millisecond
	if debounceDuration <= 0 {
		debounceDuration = 300 * time.Millisecond
	}

	ctx := &WatchContext{
		rootPath:         rootPath,
		watcher:          watcher,
		config:           config,
		gi:               gi,
		cache:            cache,
		debounceDuration: debounceDuration,
	}

	logger.Info().Msgf("Configurable watcher started on %s (recursive=%t, debounce=%dms)",
		rootPath, config.Recursive, debounceDuration/time.Millisecond)
	logger.Debug().Msgf("Monitoring %d files with filters: %v, ignore patterns: %v",
		len(cache), config.Filter, config.IgnorePatterns)
	logger.Info().Msg("Hotload is active. Press Ctrl+C to exit.")

	return runEventLoop(ctx, hook)
}
