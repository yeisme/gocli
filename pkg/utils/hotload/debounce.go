package hotload

import (
	"time"
)

// shouldDebounce checks whether we need to trigger debounced hook run.
func shouldDebounce(ctx *WatchContext) bool {
	return ctx.changeDetected
}

// armOrResetDebounce starts or resets a timer with debounce duration.
func armOrResetDebounce(ctx *WatchContext, fire func()) {
	if ctx.timer != nil {
		ctx.timer.Reset(ctx.debounceDuration)
		return
	}
	ctx.timer = time.AfterFunc(ctx.debounceDuration, fire)
}

// onDebounceFire runs when debounce timer fires: refresh cache and invoke hook.
func onDebounceFire(ctx *WatchContext, hook Func) {
	// Safety check
	if !ctx.changeDetected {
		return
	}

	logger.Info().Msg("Debounced change detected. Re-scanning state and triggering hook...")

	// Reset throttled log counters if they grew too much.
	logEventCountMutex.Lock()
	for key := range logEventCountMap {
		if logEventCountMap[key] > 100 {
			logEventCountMap[key] = 0
		}
	}
	logEventCountMutex.Unlock()

	// Rebuild state cache to ensure consistency
	newCache, err := newWatcherWithState(ctx.rootPath, ctx.config.Recursive)
	if err != nil {
		logger.Error().Msgf("Failed to update state cache after change: %v", err)
	} else {
		ctx.cache = newCache
		logger.Debug().Msgf("State cache updated with %d files", len(ctx.cache))
	}

	hook()

	// reset flags and timer
	ctx.changeDetected = false
	ctx.timer = nil
}

// stringContains is a tiny helper to avoid importing strings in multiple files.
func stringContains(s, sub string) bool {
	// minimal inline contains to avoid extra import churn; Go's inlined range
	// an import of strings would be more readable, but we encapsulate to keep deps local
	// fall back to strings.Contains if later preferred.
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
