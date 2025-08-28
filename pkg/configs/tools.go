package configs

import "github.com/spf13/viper"

// ToolsConfig represents the configuration for tools.
type ToolsConfig struct {
	Deps   []Tool `mapstructure:"deps,omitempty" json:"deps,omitempty" jsonschema:"title=Deps,description=Project local dependency tools list,uniqueItems"` // 依赖工具
	Global []Tool `mapstructure:"global,omitempty" json:"global,omitempty" jsonschema:"title=Global,description=Global (user-wide) tools list,uniqueItems"` // 全局工具

	// Go CLI 工具路径
	GoCLIToolsPath string `mapstructure:"path,omitempty" json:"path,omitempty" jsonschema:"title=Path,description=Root directory storing installed tools (may include env vars)"`
}

// Tool represents a single tool configuration.
type Tool struct {
	Type string `mapstructure:"type" json:"type,omitempty" jsonschema:"title=Type,description=Tool install type: go|clone|git,enum=go,enum=clone,enum=git"` // 工具类型：go | clone | git（同义）
	// 兼容：go 类型的安装命令，如 "go install module@ver"
	Cmd string `mapstructure:"cmd,omitempty" json:"cmd,omitempty" jsonschema:"title=Cmd,description=Legacy install command string (go install ...)"`

	// go 类型直接指定模块路径（优先于 Cmd 的解析）例如：
	// module: github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v1.59.1
	Module string `mapstructure:"module,omitempty" json:"module,omitempty" jsonschema:"title=Module,description=Go module path with optional @version (preferred over cmd),nullable"`

	// clone/git 类型：仓库地址，支持 URL#ref 语法
	CloneURL string `mapstructure:"clone,omitempty" json:"clone,omitempty" jsonschema:"title=CloneURL,description=Repository URL (optionally with #ref),nullable"`
	// 构建方式：make | goreleaser（默认 make）
	Build string `mapstructure:"build,omitempty" json:"build,omitempty" jsonschema:"title=Build,description=Build method for clone/git tool: make|goreleaser,nullable"`
	// make 目标名
	MakeTarget string `mapstructure:"make_target,omitempty" json:"make_target,omitempty" jsonschema:"title=MakeTarget,description=Make target name (empty for default),nullable"`
	// 在仓库中的子目录进行构建
	WorkDir string `mapstructure:"workdir,omitempty" json:"workdir,omitempty" jsonschema:"title=WorkDir,description=Sub directory inside repository to run build,nullable"`
	// 构建产物所在目录（相对 WorkDir/仓库根）；为空则按构建器默认（goreleaser -> dist）
	BinDirs []string `mapstructure:"bin,omitempty" json:"bin,omitempty" jsonschema:"title=BinDirs,description=Relative directories where build artifacts are produced,uniqueItems,nullable"`
	// 额外构建环境变量，如 CGO_ENABLED=1
	Env []string `mapstructure:"env,omitempty" json:"env,omitempty" jsonschema:"title=Env,description=Extra build environment variables KEY=VALUE,uniqueItems,nullable"`
	// goreleaser 配置文件路径
	GoreleaserConfig string `mapstructure:"goreleaser_config,omitempty" json:"goreleaser_config,omitempty" jsonschema:"title=GoreleaserConfig,description=Path to goreleaser config file,nullable"`
	// 输出二进制文件名（可选）
	BinaryName string `mapstructure:"binary_name,omitempty" json:"binary_name,omitempty" jsonschema:"title=BinaryName,description=Override output binary name,nullable"`
	// 是否递归克隆子模块
	RecurseSubmodules bool `mapstructure:"recurse_submodules,omitempty" json:"recurse_submodules,omitempty" jsonschema:"title=RecurseSubmodules,description=Clone git submodules recursively"`
	// 预设构建模式
	ReleaseBuild bool `mapstructure:"release_build,omitempty" json:"release_build,omitempty" jsonschema:"title=ReleaseBuild,description=Enable release build mode (may enable optimizations)"`
	DebugBuild   bool `mapstructure:"debug_build,omitempty" json:"debug_build,omitempty" jsonschema:"title=DebugBuild,description=Enable debug build mode (extra symbols, no optimizations)"`
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
	viper.SetDefault("tools.path", "~/.gocli/tools")
}
