package configs

import (
	"path/filepath"

	"github.com/spf13/viper"
)

type PluginConfig struct {
	DirPath string `mapstructure:"path"` // 插件目录路径
}

func setPluginConfigDefaults() {
	viper.SetDefault("plugin.enabled", true)
	viper.SetDefault("plugin.path", filepath.Join(".gocli", "plugins"))
}
