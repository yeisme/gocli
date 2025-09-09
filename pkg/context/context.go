// Package context provides the core application context for the gocli application.
// It includes configuration management, logging, and other shared resources.
package context

import (
	"context"

	"github.com/spf13/viper"
	"github.com/yeisme/gocli/pkg/configs"
	"github.com/yeisme/gocli/pkg/utils/log"
)

// GocliContext represents the context for the gocli application.
type GocliContext struct {
	Context context.Context
	Config  *configs.Config // 应用配置
	Logger  log.Logger      // 日志记录器
	Viper   *viper.Viper
}

// GlobalFlags holds the global flags for the application
type GlobalFlags struct {
	// ConfigPath is the path to the config file
	ConfigPath string
	// Debug enables debug mode
	Debug bool
	// Verbose enables verbose output
	Verbose bool
	// Quiet enables quiet mode
	Quiet bool
	// CPUProfile is the path to the CPU profile file
	CPUProfile string
	// Trace is the path to the execution trace file
	Trace string
	// VersionEnable enables version output
	VersionEnable bool
}

// InitGocliContext initializes the GocliContext with the provided configuration path.
func InitGocliContext(configPath string, debug, verbose, quiet bool) *GocliContext {
	ctx := context.Background()
	config, err := configs.LoadConfig(configPath)
	if err != nil {
		panic(err)
	}

	if debug {
		config.App.Debug = debug
	}
	if verbose {
		config.App.Verbose = verbose
	}
	if quiet {
		config.App.Quiet = quiet
	}

	logger := log.InitLogger(ctx, &config.Log, &config.App)

	return &GocliContext{
		Context: ctx,
		Config:  config,
		Logger:  logger,
		Viper:   configs.GetViperInstance(),
	}
}
