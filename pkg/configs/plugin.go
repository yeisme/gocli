package configs

import (
	"path/filepath"

	"github.com/spf13/viper"
)

// PluginConfig represents the configuration for plugins.
type PluginConfig struct {
	Enabled bool   `mapstructure:"enabled" jsonschema:"title=Enabled,description=Enable or disable the plugin system"`
	DirPath string `mapstructure:"path" jsonschema:"title=Path,description=Directory where plugins are stored"` // 插件目录路径
}

func setPluginConfigDefaults() {
	viper.SetDefault("plugin.enabled", true)
	viper.SetDefault("plugin.path", filepath.Join(".gocli", "plugins"))
}
