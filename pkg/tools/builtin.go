package tools

import (
	"bytes"
	_ "embed"
	"maps"
	"os"
	"sync"

	"github.com/spf13/viper"
)

type (
	// BuiltinToolsInfo 定义了内置工具的信息
	BuiltinToolsInfo struct {
		Name string `mapstructure:"name" json:"name" jsonschema:"title=Name,description=Tool identifier"`
		// URL: go install 的模块路径（可带 @version），与 CloneURL 互斥；保留兼容
		URL string `mapstructure:"url" json:"url,omitempty" jsonschema:"description=Go install module path (may include @version); mutually exclusive with CloneURL,nullable"`
		// CloneURL: 通过 git clone 构建时使用的仓库地址，可带 #ref（tag/branch/commit）
		CloneURL string `mapstructure:"clone_url" json:"clone_url,omitempty" jsonschema:"description=Repository URL used for git clone; may include #ref (tag/branch/commit),nullable"`

		// Build: clone 模式下的构建方式，例如 "make"、"goreleaser"（默认 make）
		Build string `mapstructure:"build" json:"build,omitempty" jsonschema:"description=Build method for clone mode, e.g. make or goreleaser,nullable"`
		// MakeTarget: make 模式下要执行的目标名（可为空执行默认目标）
		MakeTarget string `mapstructure:"make_target" json:"make_target,omitempty" jsonschema:"description=Make target to run (empty for default),nullable"`
		// WorkDir: 在仓库中的子目录进行构建（为空则仓库根目录）
		WorkDir string `mapstructure:"work_dir" json:"work_dir,omitempty" jsonschema:"description=Subdirectory within the repository to build in (empty means repo root),nullable"`
		// BinDirs: 构建产物所在目录（相对 WorkDir/仓库根）；为空则按构建器默认（goreleaser -> dist）
		BinDirs []string `mapstructure:"bin_dirs" json:"bin_dirs,omitempty" jsonschema:"description=Directories containing build artifacts (relative to WorkDir/repo root),nullable,uniqueItems"`
		// Env: 该工具需要的额外环境变量（如 CGO_ENABLED=1）
		Env []string `mapstructure:"env" json:"env,omitempty" jsonschema:"description=Additional environment variables required by the tool (e.g., CGO_ENABLED=1),nullable,uniqueItems"`
		// GoreleaserConfig: goreleaser 构建时的配置文件路径（相对 WorkDir/仓库根）
		GoreleaserConfig string `mapstructure:"goreleaser_config" json:"goreleaser_config,omitempty" jsonschema:"description=Path to goreleaser config file (relative to WorkDir/repo root),nullable"`

		InstallType *InstallType `mapstructure:"install_type" json:"install_type,omitempty" jsonschema:"description=Platform constraints for installation; contains name/os/arch; may be null,nullable"`
		// BinaryName 指定该工具生成的二进制文件名（可选）；
		// 若为空则使用默认规则（例如 go install 的默认产物名）
		BinaryName string `mapstructure:"binary_name" json:"binary_name,omitempty" jsonschema:"description=Optional binary filename produced by the tool; if empty default rules apply (e.g., go install),nullable"`
	}

	// InstallType 定义了内置工具的安装类型
	InstallType struct {
		Name string `mapstructure:"name" json:"name,omitempty" jsonschema:"description=Install type name, e.g. go_install or clone,nullable"`
		OS   string `mapstructure:"os" json:"os,omitempty" jsonschema:"description=Restricted OS name, e.g. linux/darwin/windows,nullable"`
		Arch string `mapstructure:"arch" json:"arch,omitempty" jsonschema:"description=Restricted CPU architecture, e.g. amd64/arm64,nullable"`
	}
)

//go:embed embed_tools.json
var embedTools []byte

// BuiltinTools 内部维护工具的名称和安装路径（既可 go install，也可 clone 构建）
var BuiltinTools map[string]BuiltinToolsInfo

var builtinInitOnce sync.Once

func init() {
	// 使用 sync.Once 确保只初始化一次
	builtinInitOnce.Do(func() {
		// 尝试从嵌入的 JSON 初始化
		if len(embedTools) > 0 {
			v := viper.New()
			v.SetConfigType("json")
			if err := v.ReadConfig(bytes.NewReader(embedTools)); err == nil {
				var m map[string]BuiltinToolsInfo
				if err := v.Unmarshal(&m); err == nil && len(m) > 0 {
					BuiltinTools = m
					return
				}
			}
		}
		// 如果嵌入资源不可用或为空，则初始化为空映射，后续可通过 LoadUserTools 合并用户定义
		BuiltinTools = make(map[string]BuiltinToolsInfo)
	})
}

// LoadUserTools 从指定的 JSON 文件加载用户自定义的工具表并合并到全局 BuiltinTools 中
// 如果键存在则会覆盖已有条目
func LoadUserTools(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var m map[string]BuiltinToolsInfo
	v := viper.New()

	v.SetConfigType("json")
	v.SetConfigType("yaml")
	v.SetConfigType("yml")
	v.SetConfigType("toml")

	if err := v.ReadConfig(bytes.NewReader(data)); err != nil {
		return err
	}
	if err := v.Unmarshal(&m); err != nil {
		return err
	}
	if BuiltinTools == nil {
		BuiltinTools = make(map[string]BuiltinToolsInfo)
	}
	maps.Copy(BuiltinTools, m)
	return nil
}
