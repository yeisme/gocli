package configs

import "github.com/spf13/viper"

type Tools struct {
	Deps   []Tool `mapstructure:"deps,omitempty"`   // 依赖工具
	Global []Tool `mapstructure:"global,omitempty"` // 全局工具
}

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
}
