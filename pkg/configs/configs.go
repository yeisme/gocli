// Package configs 提供应用程序配置管理功能
package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Version int         `mapstructure:"version" jsonschema:"title=Version,description=Configuration file version,minimum=1,required"`
	Log     LogConfig   `mapstructure:"log" jsonschema:"title=Log,description=Logging related settings"`
	Env     EnvConfig   `mapstructure:"env" jsonschema:"title=Env,description=Go related environment variables (auto-detected + overrides)"`
	App     AppConfig   `mapstructure:"app" jsonschema:"title=App,description=General application behavior flags"`
	Tools   ToolsConfig `mapstructure:"tools" jsonschema:"title=Tools,description=Project and global tool installation configuration"`
	Doc     DocConfig   `mapstructure:"doc" jsonschema:"title=Doc,description=Documentation generation options"`
	Init    InitConfig  `mapstructure:"init" jsonschema:"title=Init,description=Project initialization template settings"`
}

// setDefaults 设置默认配置值
func setDefaults() {
	viper.SetDefault("version", 1)
	setLogConfigDefaults()
	setEnvConfigDefaults()
	setAppConfigDefaults()
	setToolsConfigDefaults()
	setDocConfigDefaults()
	setInitConfigDefaults()
}

var globalConfig *Config

// tryLoadConfigFiles 尝试加载不同格式的配置文件
func tryLoadConfigFiles(base string) bool {
	// 最终搜索路径列表（按优先级）
	var searchPaths []string

	// 1. 指定 base（模块根或显式目录）
	if base != "" {
		searchPaths = append(searchPaths, base, filepath.Join(base, "configs"))
	}

	// 2. 当前工作目录向上回溯，直到文件系统根或已经包含 base
	if cwd, err := os.Getwd(); err == nil {
		// 向上遍历
		cur := cwd
		for {
			// 如果已经添加过则停止
			already := slices.Contains(searchPaths, cur)
			if !already {
				searchPaths = append(searchPaths, cur, filepath.Join(cur, "configs"))
			}
			parent := filepath.Dir(cur)
			if parent == cur { // 到根目录
				break
			}
			cur = parent
			// 如果 base 非空且我们已回溯到 base 之上，可以继续直到根，确保向上所有层都能检测
		}
	}

	// 3. 预定义全局搜索路径（HOME 等）
	searchPaths = append(searchPaths, GetConfigSearchPaths()...)

	// 去重（保持顺序）
	dedup := make([]string, 0, len(searchPaths))
	seen := make(map[string]struct{})
	for _, p := range searchPaths {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		dedup = append(dedup, p)
	}
	searchPaths = dedup

	configNames := []string{".gocli", "gocli"}
	extensions := []string{"yaml", "yml", "json", "toml"}

	for _, dir := range searchPaths {
		for _, name := range configNames {
			for _, ext := range extensions {
				configFile := filepath.Join(dir, name+"."+ext)
				if strings.Contains(configFile, "$") {
					configFile = os.ExpandEnv(configFile)
				}
				if fi, err := os.Stat(configFile); err == nil && !fi.IsDir() {
					viper.SetConfigFile(configFile)
					return true
				}
			}
		}
	}
	return false
}

// GetConfigSearchPaths 返回配置和资源搜索路径列表，供其他包复用
func GetConfigSearchPaths() []string {
	searchPaths := []string{
		".",
		"./configs",
		"$HOME",
		"$HOME/.config",
		"$HOME/.config/gocli",
	}

	if runtime.GOOS == "windows" {
		searchPaths = append(searchPaths,
			"$USERPROFILE",
			"$APPDATA/gocli",
		)
	} else {
		searchPaths = append(searchPaths, "/etc/gocli")
	}

	return searchPaths
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	// 设置默认值
	setDefaults()

	if configPath != "" {
		// 使用指定的配置文件路径
		viper.SetConfigFile(configPath)
	} else if moduleRoot := GetModuleRoot(""); moduleRoot != "" {
		// 使用模块根目录进行搜索（支持在子目录中执行命令）
		tryLoadConfigFiles(moduleRoot)
	} else {
		// 尝试查找多种格式的配置文件
		tryLoadConfigFiles("")
	}

	// 设置环境变量前缀
	viper.SetEnvPrefix("GOCLI")
	viper.AutomaticEnv()

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

	// 获取所在的配置文件的目录（如果没有加载到配置文件则不修改）
	cfgFile := viper.ConfigFileUsed()
	if cfgFile != "" {
		cfgDir := filepath.Dir(cfgFile)
		if config.Log.FilePath == "" {
			config.Log.FilePath = filepath.Join(cfgDir, ".gocli/gocli.log")
		} else if !filepath.IsAbs(config.Log.FilePath) {
			config.Log.FilePath = filepath.Join(cfgDir, config.Log.FilePath)
		}
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

// GetViperInstance 返回当前的 viper 实例
func GetViperInstance() *viper.Viper {
	return viper.GetViper()
}
