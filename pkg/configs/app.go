package configs

import (
	"github.com/spf13/viper"
)

// AppConfig 应用配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Debug   bool   `mapstructure:"debug"`
	Verbose bool   `mapstructure:"verbose"`
	Quiet   bool   `mapstructure:"quiet"`
}

func setAppConfigDefaults() {
	viper.SetDefault("app.name", "gocli")
	viper.SetDefault("app.debug", false)
	viper.SetDefault("app.verbose", false)
	viper.SetDefault("app.quiet", false)
}
