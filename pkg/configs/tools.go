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
	Type string `mapstructure:"type"` // 工具类型：go | clone | git（同义）
	// 兼容：go 类型的安装命令，如 "go install module@ver"
	Cmd string `mapstructure:"cmd,omitempty"`

	// go 类型直接指定模块路径（优先于 Cmd 的解析）。例如：
	// module: github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v1.59.1
	Module string `mapstructure:"module,omitempty"`

	// clone/git 类型：仓库地址，支持 URL#ref 语法
	CloneURL string `mapstructure:"clone,omitempty"`
	// 构建方式：make | goreleaser（默认 make）
	Build string `mapstructure:"build,omitempty"`
	// make 目标名
	MakeTarget string `mapstructure:"make_target,omitempty"`
	// 在仓库中的子目录进行构建
	WorkDir string `mapstructure:"workdir,omitempty"`
	// 构建产物所在目录（相对 WorkDir/仓库根）；为空则按构建器默认（goreleaser -> dist）
	BinDirs []string `mapstructure:"bin,omitempty"`
	// 额外构建环境变量，如 CGO_ENABLED=1
	Env []string `mapstructure:"env,omitempty"`
	// goreleaser 配置文件路径
	GoreleaserConfig string `mapstructure:"goreleaser_config,omitempty"`
	// 输出二进制文件名（可选）
	BinaryName string `mapstructure:"binary_name,omitempty"`
	// 是否递归克隆子模块
	RecurseSubmodules bool `mapstructure:"recurse_submodules,omitempty"`
	// 预设构建模式
	ReleaseBuild bool `mapstructure:"release_build,omitempty"`
	DebugBuild   bool `mapstructure:"debug_build,omitempty"`
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
