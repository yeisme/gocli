package hotload

import (
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/yeisme/gocli/pkg/configs"
)

// baseDirWatcherWithConfig 是简易的协调器，用于将 watcher、缓存和过滤器连接起来并启动事件循环.
func baseDirWatcherWithConfig(rootPath string, config configs.HotloadConfig, hook Func) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建 watcher 失败: %w", err)
	}
	defer func() {
		if cerr := watcher.Close(); cerr != nil {
			logger.Error().Msgf("关闭 watcher 失败: %v", cerr)
		}
	}()

	// 初始文件状态缓存
	cache, err := initializeFileStateCache(rootPath, config.Recursive)
	if err != nil {
		return err
	}

	// 在启用时加载 .gitignore
	gi, err := loadGitIgnore(rootPath, config.GitIgnore)
	if err != nil {
		return err
	}

	// 注册要监视的目录
	if config.Recursive {
		if err := addDirectoriesToWatcher(watcher, rootPath, config, gi); err != nil {
			return err
		}
	} else {
		if err := watcher.Add(rootPath); err != nil {
			return fmt.Errorf("将根路径 '%s' 添加到 watcher 失败: %w", rootPath, err)
		}
	}

	// 防抖时长
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

	logger.Info().Msgf("已在 %s 启动可配置的 watcher (recursive=%t, debounce=%dms)",
		rootPath, config.Recursive, debounceDuration/time.Millisecond)
	logger.Debug().Msgf("监视 %d 个文件，过滤器: %v，忽略模式: %v",
		len(cache), config.Filter, config.IgnorePatterns)
	logger.Info().Msg("Hotload 已启动.按 Ctrl+C 退出.")

	return runEventLoop(ctx, hook)
}
