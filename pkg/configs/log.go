package configs

import "github.com/spf13/viper"

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level" jsonschema:"title=Level,description=Log level: trace|debug|info|warn|error|fatal|panic,enum=trace,enum=debug,enum=info,enum=warn,enum=error,enum=fatal,enum=panic"` // 日志级别
	JSON       bool   `mapstructure:"json" jsonschema:"title=JSON,description=Output logs in JSON format"`                                                                                                      // 是否使用 JSON 格式输出
	Mode       string `mapstructure:"mode" jsonschema:"title=Mode,description=Log output mode: console|file|both,enum=console,enum=file,enum=both"`                                                             // 输出模式
	FilePath   string `mapstructure:"file_path" jsonschema:"title=FilePath,description=Log file path when mode includes file,nullable"`                                                                         // 文件路径
	MaxSize    int    `mapstructure:"max_size" jsonschema:"title=MaxSize,description=Maximum log file size in MB before rotation,minimum=1"`                                                                    // 日志文件最大大小（MB）
	MaxBackups int    `mapstructure:"max_backups" jsonschema:"title=MaxBackups,description=Number of rotated log files to retain,minimum=0"`                                                                    // 保留的备份文件数量
	MaxAge     int    `mapstructure:"max_age" jsonschema:"title=MaxAge,description=Maximum age in days to retain old log files,minimum=0"`                                                                      // 文件保留天数
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
