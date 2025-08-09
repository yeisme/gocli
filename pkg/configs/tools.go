package configs

import "github.com/spf13/viper"

// ToolsConfig represents the configuration for tools.
type ToolsConfig struct {
	Deps   []Tool `mapstructure:"deps,omitempty"`   // 依赖工具
	Global []Tool `mapstructure:"global,omitempty"` // 全局工具

	// Go CLI 工具路径
	GoCLIToolsPath string `mapstructure:"path,omitempty"`
}

// Tool represents a single tool configuration.
type Tool struct {
	Type string `mapstructure:"type"` // 工具类型，如 go, npm 等
	Cmd  string `mapstructure:"cmd"`  // 工具的安装命令
}

func setToolsConfigDefaults() {
	viper.SetDefault("tools.deps", []Tool{
		{
			Type: "go",
			Cmd:  "go install github.com/go-task/task/v3/cmd/task@latest",
		},
	})
	viper.SetDefault("tools.global", []Tool{
		{
			Type: "go",
			Cmd:  "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest",
		},
	})
	viper.SetDefault("tools.path", "$HOME/.gocli/tools")
}
