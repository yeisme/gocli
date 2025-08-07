// Package log 提供全局日志记录器的初始化和获取功能
// 使用 zerolog 作为日志库，支持多种输出模式（控制台、文件、两者）
// 支持不同的日志级别（Trace、Debug、Info、Warn、Error、Fatal、Panic）
package log

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/yeisme/gocli/pkg/configs"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 定义全局日志记录器类型
// 使用 *zerolog.Logger 作为日志记录器类型
type Logger = *zerolog.Logger

// globalLogger 全局日志记录器实例
// 这个变量在 InitLogger 中被初始化
// 并在 GetLogger 中被访问
// 这样可以确保在应用程序的任何地方都能获取到一致的日志记录
var globalLogger Logger

// InitLogger 初始化日志记录器
func InitLogger(ctx context.Context, config *configs.LogConfig, appConfig *configs.AppConfig) Logger {
	// 优先级：quiet > debug > verbose > config.Level
	if appConfig.Quiet {
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
		logger := zerolog.New(io.Discard)
		globalLogger = &logger
		log.Logger = logger
		return &logger
	} else if appConfig.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else if appConfig.Verbose {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		level := parseLogLevel(config.Level)
		zerolog.SetGlobalLevel(level)
	}

	// 创建输出目标
	var writers []io.Writer

	// 根据模式配置输出
	switch strings.ToLower(config.Mode) {
	case "console":
		writers = append(writers, createConsoleWriter(config.JSON))
	case "file":
		writers = append(writers, createFileWriter(config))
	case "both":
		writers = append(writers, createConsoleWriter(config.JSON))
		writers = append(writers, createFileWriter(config))
	default:
		// 默认输出到控制台
		writers = append(writers, createConsoleWriter(config.JSON))
	}

	// 创建多重写入器
	var output io.Writer
	if len(writers) == 1 {
		output = writers[0]
	} else {
		output = io.MultiWriter(writers...)
	}

	var logger zerolog.Logger

	// 创建日志记录器
	if appConfig.Debug {
		logger = zerolog.New(output).With().Caller().
			Str("app", appConfig.Name).
			Ctx(ctx).Timestamp().Logger()
	} else if appConfig.Verbose {
		logger = zerolog.New(output).With().
			Str("app", appConfig.Name).
			Ctx(ctx).Timestamp().Logger()
	} else {
		logger = zerolog.New(output).With().Timestamp().Logger()
	}

	globalLogger = &logger
	log.Logger = logger
	return &logger
}

// createConsoleWriter 创建控制台输出写入器
func createConsoleWriter(useJSON bool) io.Writer {
	if useJSON {
		return os.Stdout
	}
	return zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	}
}

// createFileWriter 创建文件输出写入器
func createFileWriter(config *configs.LogConfig) io.Writer {
	// 确保日志目录存在
	logDir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return os.Stdout
	}

	// 使用 lumberjack 进行日志轮转
	return &lumberjack.Logger{
		Filename:   config.FilePath,
		MaxSize:    config.MaxSize,    // megabytes
		MaxBackups: config.MaxBackups, // 保留备份数量
		MaxAge:     config.MaxAge,     // days
		Compress:   true,              // 压缩旧日志文件
	}
}

// GetLogger 获取全局日志记录器
func GetLogger() Logger {
	if globalLogger == nil {
		config := configs.GetConfig()
		return InitLogger(context.Background(), &config.Log, &config.App)
	}
	return globalLogger
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// WithFields 带字段的日志记录
func WithFields(fields map[string]any) Logger {
	logger := GetLogger()
	event := logger.With()
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	result := event.Logger()
	return &result
}

// Trace 创建一个 Trace 级别的日志事件
func Trace() *zerolog.Event {
	return GetLogger().Trace()
}

// Debug 创建一个 Debug 级别的日志事件
func Debug() *zerolog.Event {
	return GetLogger().Debug()
}

// Info 创建一个 Info 级别的日志事件
func Info() *zerolog.Event {
	return GetLogger().Info()
}

// Warn 创建一个 Warn 级别的日志事件
func Warn() *zerolog.Event {
	return GetLogger().Warn()
}

// Error 创建一个 Error 级别的日志事件
func Error() *zerolog.Event {
	return GetLogger().Error()
}

// Fatal 创建一个 Fatal 级别的日志事件
func Fatal() *zerolog.Event {
	return GetLogger().Fatal()
}

// Panic 创建一个 Panic 级别的日志事件
func Panic() *zerolog.Event {
	return GetLogger().Panic()
}
