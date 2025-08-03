package context

import (
	"context"

	"github.com/spf13/viper"
	"github.com/yeisme/gocli/pkg/configs"
	"github.com/yeisme/gocli/pkg/utils"
)

type GocliContext struct {
	context.Context
	Config *configs.Config // 应用配置
	Logger utils.Logger    // 日志记录器
	Viper *viper.Viper
}

func InitGocliContext(configPath string) *GocliContext {
	ctx := context.Background()
	config, err := configs.LoadConfig(configPath)
	if err != nil {
		panic(err)
	}

	logger := utils.InitLogger(&config.Log, &config.App, ctx)

	return &GocliContext{
		Context: ctx,
		Config:  config,
		Logger:  logger,
		Viper: configs.GetViperInstance(),
	}
}
