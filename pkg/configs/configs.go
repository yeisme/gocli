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
	Version string       `mapstructure:"version"`
	Log     LogConfig    `mapstructure:"log"`
	Env     EnvConfig    `mapstructure:"env"`
	App     AppConfig    `mapstructure:"app"`
	Tools   ToolsConfig  `mapstructure:"tools"`
	Plugin  PluginConfig `mapstructure:"plugin"`
	Doc     DocConfig    `mapstructure:"doc"`
}

// setDefaults 设置默认配置值
func setDefaults() {
	viper.SetDefault("version", "1.0")
	setLogConfigDefaults()
	setEnvConfigDefaults()
	setAppConfigDefaults()
	setToolsConfigDefaults()
	setPluginConfigDefaults()
	setDocConfigDefaults()
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
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 应用环境变量
	config.Env.ApplyEnvVars()

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

// GetViperInstance 返回当前的 viper 实例
func GetViperInstance() *viper.Viper {
	return viper.GetViper()
}
