package parse

import (
	"os"

	"github.com/goccy/go-yaml"
	"github.com/yeisme/gocli/pkg/types"
)

var (
	projectConfig *types.Config
	userConfig    *types.Config
)

func ParseYAML(data []byte) (map[string]any, error) {
	var result map[string]any
	err := yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ParseConfig parses YAML configuration file to Config struct
func ParseConfig(data []byte) (*types.Config, error) {
	var config types.Config
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ParseConfigFromFile parses YAML configuration from file path
func ParseConfigFromFile(filePath string) (*types.Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return ParseConfig(data)
}

// SetProjectConfig sets project configuration
func SetProjectConfig(config *types.Config) {
	projectConfig = config
}

// GetProjectConfig gets project configuration
func GetProjectConfig() *types.Config {
	return projectConfig
}

// SetUserConfig sets user configuration
func SetUserConfig(config *types.Config) {
	userConfig = config
}

// GetUserConfig gets user configuration
func GetUserConfig() *types.Config {
	return userConfig
}
