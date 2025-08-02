// Package utils 提供通用工具函数，包括日志记录功能
package utils

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

type Logger = *zerolog.Logger

var globalLogger Logger

// InitLogger 初始化日志记录器
func InitLogger(config *configs.LogConfig, appConfig *configs.AppConfig, ctx context.Context) Logger {
	if appConfig.Quiet && appConfig.Verbose {
		panic("Cannot set both quiet and verbose modes")
	}

	if appConfig.Verbose {
		// 如果是 Debug 模式，设置日志级别为 Debug
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else if appConfig.Quiet {
		// 如果是安静模式，禁止所有日志输出
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	} else if appConfig.Debug {
		// 如果是 Debug 模式，设置日志级别为 Debug
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		// 设置日志级别
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
		return InitLogger(&config.Log, &config.App, context.Background())
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

func Trace() *zerolog.Event {
	return GetLogger().Trace()
}

func Debug() *zerolog.Event {
	return GetLogger().Debug()
}

func Info() *zerolog.Event {
	return GetLogger().Info()
}

func Warn() *zerolog.Event {
	return GetLogger().Warn()
}

func Error() *zerolog.Event {
	return GetLogger().Error()
}

func Fatal() *zerolog.Event {
	return GetLogger().Fatal()
}

func Panic() *zerolog.Event {
	return GetLogger().Panic()
}
