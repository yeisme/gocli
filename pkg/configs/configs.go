// Package configs 提供应用程序配置管理功能
package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Version string    `mapstructure:"version"`
	Log     LogConfig `mapstructure:"log"`
	App     AppConfig `mapstructure:"app"`
}

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

// AppConfig 应用配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Debug   bool   `mapstructure:"debug"`
	Verbose bool   `mapstructure:"verbose"`
	Quiet   bool   `mapstructure:"quiet"` // 是否安静模式，禁止所有日志输出
}

// setDefaults 设置默认配置值
func setDefaults() {
	viper.SetDefault("version", "1.0")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.json", false)
	viper.SetDefault("log.mode", "console")
	viper.SetDefault("log.file_path", ".gocli/gocli.log")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_age", 28)
	viper.SetDefault("app.name", "gocli")
	viper.SetDefault("app.debug", false)
	viper.SetDefault("app.verbose", false)
	viper.SetDefault("app.quiet", false)
}

var globalConfig *Config

// tryLoadConfigFiles 尝试加载不同格式的配置文件
func tryLoadConfigFiles() bool {
	// 配置文件搜索路径
	searchPaths := []string{
		".",
		"./configs",
		"$HOME",
		"$HOME/.config",
		"$HOME/.config/gocli",
	}

	// Windows 特殊路径
	if runtime.GOOS == "windows" {
		searchPaths = append(searchPaths,
			"$USERPROFILE",
			"$APPDATA/gocli",
		)
	} else {
		searchPaths = append(searchPaths, "/etc/gocli")
	}

	// 配置文件名和扩展名的组合
	configNames := []string{".gocli", "gocli"}
	extensions := []string{"yaml", "yml", "json", "toml"}

	for _, path := range searchPaths {
		for _, name := range configNames {
			for _, ext := range extensions {
				configFile := filepath.Join(path, name+"."+ext)

				// 展开环境变量
				if strings.Contains(configFile, "$") {
					configFile = os.ExpandEnv(configFile)
				}

				if _, err := os.Stat(configFile); err == nil {
					viper.SetConfigFile(configFile)
					return true
				}
			}
		}
	}

	return false
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	if configPath != "" {
		// 使用指定的配置文件路径
		viper.SetConfigFile(configPath)
	} else {
		// 尝试查找多种格式的配置文件
		tryLoadConfigFiles()
	}

	// 设置环境变量前缀
	viper.SetEnvPrefix("GOCLI")
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		} else {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 确保日志目录存在
	if config.Log.Mode == "file" || config.Log.Mode == "both" {
		logDir := filepath.Dir(config.Log.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %w", err)
		}
	}

	globalConfig = &config
	return &config, nil
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	if globalConfig == nil {
		config, err := LoadConfig("")
		if err != nil {
			panic(fmt.Sprintf("无法加载配置: %v", err))
		}
		return config
	}
	return globalConfig
}
