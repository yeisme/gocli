package configs

import (
	"github.com/spf13/viper"
)

// AppConfig 应用配置
type AppConfig struct {
	Name    string        `mapstructure:"name"`
	Debug   bool          `mapstructure:"debug"`
	Verbose bool          `mapstructure:"verbose"`
	Quiet   bool          `mapstructure:"quiet"`
	Hotload HotloadConfig `mapstructure:"hotload"`
}

// HotloadConfig 热加载配置
type HotloadConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	Dir            string   `mapstructure:"dir"`
	Filter         []string `mapstructure:"filter"`
	Recursive      bool     `mapstructure:"recursive"`
	Debounce       int      `mapstructure:"debounce"`        // 防抖时间，毫秒
	IgnorePatterns []string `mapstructure:"ignore_patterns"` // 忽略的文件模式
	GitIgnore      bool     `mapstructure:"git_ignore"`      // 是否使用 .gitignore 文件
}

func setAppConfigDefaults() {
	viper.SetDefault("app.name", "gocli")
	viper.SetDefault("app.debug", false)
	viper.SetDefault("app.verbose", false)
	viper.SetDefault("app.quiet", false)

	// 热加载配置默认值
	viper.SetDefault("app.hotload.enabled", false)
	viper.SetDefault("app.hotload.dir", ".")
	viper.SetDefault("app.hotload.filter", []string{"*.go", "*.mod", "tpl", "tpl"})
	viper.SetDefault("app.hotload.recursive", true)
	viper.SetDefault("app.hotload.debounce", 300) // 毫秒
	viper.SetDefault("app.hotload.ignore_patterns", []string{
		"*.tmp",
		"*.swp",
		"*.log",
		"tmp/*",
		"vendor/*",
		".git/*",
		"node_modules/*",
	})
	viper.SetDefault("app.hotload.git_ignore", true) // 默认使用 .gitignore
}
