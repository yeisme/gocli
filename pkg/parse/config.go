package parse

import (
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

// ProjectConfig returns the project configuration.
// If the user flag is set, it returns the user configuration instead.
func ProjectConfig() *types.Config {
	if utils.IsUser() {
		if userConfig == nil {
			userConfig, _ = ParseConfigFromFile(utils.UserConfigPath())
		}
		return userConfig
	}
	return projectConfig
}

func UserConfig() *types.Config {
	if userConfig == nil {
		userConfig, _ = ParseConfigFromFile(utils.UserConfigPath())
	}
	return userConfig
}
