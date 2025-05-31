package parse

import (
	"os"

	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

// ProjectConfig returns the project configuration.
// If the user flag is set, it returns the user configuration instead.
func ProjectConfig() *types.Config {
	if utils.IsUser() {
		return UserConfig()
	}

	// If project config is already loaded, return it
	if projectConfig != nil {
		return projectConfig
	}

	// Try to load project config
	configPath := utils.ProjectConfigPath(utils.GetConfigFile())
	if _, err := os.Stat(configPath); err == nil {
		config, err := ParseConfigFromFile(configPath)
		if err == nil {
			SetProjectConfig(config)
			return config
		}
		utils.Debug("Failed to parse project config: %v", err)
	}

	return nil
}

func UserConfig() *types.Config {
	if userConfig != nil {
		return userConfig
	}

	configPath := utils.UserConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		config, err := ParseConfigFromFile(configPath)
		if err == nil {
			SetUserConfig(config)
			return config
		}
		utils.Debug("Failed to parse user config: %v", err)
	}

	return nil
}
