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

// InitGocliContext initializes the GocliContext with the provided configuration path.
func InitGocliContext(configPath string, debug, verbose, quiet bool) *GocliContext {
	ctx := context.Background()
	config, err := configs.LoadConfig(configPath)
	if err != nil {
		panic(err)
	}

	// config is guaranteed non-nil when err == nil
	config.App.Debug = debug
	config.App.Verbose = verbose
	config.App.Quiet = quiet

	logger := log.InitLogger(ctx, &config.Log, &config.App)

	return &GocliContext{
		Context: ctx,
		Config:  config,
		Logger:  logger,
		Viper:   configs.GetViperInstance(),
	}
}
