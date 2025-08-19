package tools

// BuildContext 提供构建时的上下文信息
type BuildContext struct {
	RepoDir     string   // 仓库根目录
	BuildDir    string   // 实际执行构建的目录（可能等于 RepoDir 或其子目录）
	AbsBase     string   // 基础安装目录（用于默认 GOBIN 等）
	RepoURL     string   // 仓库 URL
	ResolvedRef string   // 实际检出的 ref（可能是具体 tag）
	DisplayRef  string   // 展示用 ref（如 latest）
	Env         []string // 环境变量
	Verbose     bool
}

// BuildParams 是通用的构建参数载体
type BuildParams struct {
	// make 相关
	MakeTarget string
	// goreleaser 相关与通用附加参数
	GoreleaserConfig string
	BuildArgs        []string
}

// BuildRunner 定义不同构建方式的统一接口
type BuildRunner interface {
	// Name 返回构建器名称（例如 "make"、"goreleaser"）
	Name() string
	// DefaultBinDirs 返回该构建器典型的产物输出目录（相对 BuildDir）
	DefaultBinDirs() []string
	// Build 执行构建，返回标准输出/错误的合并文本
	Build(ctx BuildContext, params BuildParams) (string, error)
}

var buildRunners = map[string]BuildRunner{}

// RegisterBuildRunner 注册一个构建器
func RegisterBuildRunner(r BuildRunner) {
	if r == nil {
		return
	}
	buildRunners[r.Name()] = r
}

// GetBuildRunner 获取构建器
func GetBuildRunner(name string) (BuildRunner, bool) {
	r, ok := buildRunners[name]
	return r, ok
}
