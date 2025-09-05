package hotload

import (
	"time"
)

// shouldDebounce 判断是否需要触发防抖（debounce）钩子执行.
func shouldDebounce(ctx *WatchContext) bool {
	return ctx.changeDetected
}

// armOrResetDebounce 启动或重置一个基于配置防抖时长的定时器.
func armOrResetDebounce(ctx *WatchContext, fire func()) {
	if ctx.timer != nil {
		ctx.timer.Reset(ctx.debounceDuration)
		return
	}
	ctx.timer = time.AfterFunc(ctx.debounceDuration, fire)
}

// onDebounceFire 在防抖定时器触发时运行：刷新状态缓存并调用钩子.
func onDebounceFire(ctx *WatchContext, hook Func) {
	// 安全检查
	if !ctx.changeDetected {
		return
	}

	logger.Info().Msg("检测到防抖后的变更.重新扫描状态并触发钩子...")

	// 如果日志计数过大则重置计数器
	logEventCountMutex.Lock()
	for key := range logEventCountMap {
		if logEventCountMap[key] > 100 {
			logEventCountMap[key] = 0
		}
	}
	logEventCountMutex.Unlock()

	// 重新构建状态缓存以确保一致性
	newCache, err := newWatcherWithState(ctx.rootPath, ctx.config.Recursive)
	if err != nil {
		logger.Error().Msgf("变更后更新状态缓存失败: %v", err)
	} else {
		ctx.cache = newCache
		logger.Debug().Msgf("状态缓存已更新，包含 %d 个文件", len(ctx.cache))
	}

	hook()

	// 重置标记和定时器
	ctx.changeDetected = false
	ctx.timer = nil
}

// stringContains 是一个小工具，避免在多个文件中引入 strings 包.
func stringContains(s, sub string) bool {
	// 使用最小化的内联比较以减少依赖；如有需要可改为 strings.Contains
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
