package configs

import (
	"github.com/spf13/viper"
)

// AppConfig 应用配置
type AppConfig struct {
	Name    string        `mapstructure:"name" jsonschema:"title=Name,description=Application display name"`
	Debug   bool          `mapstructure:"debug" jsonschema:"title=Debug,description=Enable debug mode (more verbose internal logging)"`
	Verbose bool          `mapstructure:"verbose" jsonschema:"title=Verbose,description=Enable verbose output for commands"`
	Quiet   bool          `mapstructure:"quiet" jsonschema:"title=Quiet,description=Suppress non-error output"`
	Hotload HotloadConfig `mapstructure:"hotload" jsonschema:"title=Hotload,description=File watching / hot reload settings"`
}

// HotloadConfig 热加载配置
type HotloadConfig struct {
	Enabled        bool     `mapstructure:"enabled" jsonschema:"title=Enabled,description=Enable hot file watching"`
	Dir            string   `mapstructure:"dir" jsonschema:"title=Dir,description=Root directory to watch"`
	Filter         []string `mapstructure:"filter" jsonschema:"title=Filter,description=Glob patterns to include for watching,uniqueItems"`
	Recursive      bool     `mapstructure:"recursive" jsonschema:"title=Recursive,description=Watch directories recursively"`
	Debounce       int      `mapstructure:"debounce" jsonschema:"title=Debounce,description=Event debounce time in milliseconds,minimum=0"`    // 防抖时间
	IgnorePatterns []string `mapstructure:"ignore_patterns" jsonschema:"title=IgnorePatterns,description=Glob patterns to ignore,uniqueItems"` // 忽略的文件模式
	GitIgnore      bool     `mapstructure:"git_ignore" jsonschema:"title=GitIgnore,description=Honor .gitignore exclusions"`                   // 是否使用 .gitignore 文件
}

func setAppConfigDefaults() {
	viper.SetDefault("app.name", "gocli")
	viper.SetDefault("app.debug", false)
	viper.SetDefault("app.verbose", false)
	viper.SetDefault("app.quiet", false)

	// 热加载配置默认值
	viper.SetDefault("app.hotload.enabled", false)
	viper.SetDefault("app.hotload.dir", ".")
	viper.SetDefault("app.hotload.filter", []string{"*.go", "*.mod", "*.tpl"})
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
