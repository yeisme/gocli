package configs

import "github.com/spf13/viper"

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`       // 日志级别: trace, debug, info, warn, error, fatal, panic
	JSON       bool   `mapstructure:"json"`        // 是否使用 JSON 格式输出
	Mode       string `mapstructure:"mode"`        // 输出模式: console, file, both
	FilePath   string `mapstructure:"file_path"`   // 文件路径（当 mode 为 file 或 both 时使用）
	MaxSize    int    `mapstructure:"max_size"`    // 日志文件最大大小（MB）
	MaxBackups int    `mapstructure:"max_backups"` // 保留的备份文件数量
	MaxAge     int    `mapstructure:"max_age"`     // 文件保留天数
}

func setLogConfigDefaults() {
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.json", false)
	viper.SetDefault("log.mode", "console")
	viper.SetDefault("log.file_path", ".gocli/gocli.log")
	viper.SetDefault("log.max_size", 100)  // MB
	viper.SetDefault("log.max_backups", 3) // 保留的备份文件数量
	viper.SetDefault("log.max_age", 28)    // 文件保留天数
}
