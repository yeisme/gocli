package tools

type (
	// BuiltinToolsInfo 定义了内置工具的信息
	BuiltinToolsInfo struct {
		Name string
		// URL: go install 的模块路径（可带 @version），与 CloneURL 互斥；保留兼容
		URL string
		// CloneURL: 通过 git clone 构建时使用的仓库地址，可带 #ref（tag/branch/commit）
		CloneURL string

		// Build: clone 模式下的构建方式，例如 "make"、"goreleaser"（默认 make）
		Build string
		// MakeTarget: make 模式下要执行的目标名（可为空执行默认目标）
		MakeTarget string
		// WorkDir: 在仓库中的子目录进行构建（为空则仓库根目录）
		WorkDir string
		// BinDirs: 构建产物所在目录（相对 WorkDir/仓库根）；为空则按构建器默认（goreleaser -> dist）
		BinDirs []string
		// Env: 该工具需要的额外环境变量（如 CGO_ENABLED=1）
		Env []string
		// GoreleaserConfig: goreleaser 构建时的配置文件路径（相对 WorkDir/仓库根）
		GoreleaserConfig string

		InstallType InstallType
		// BinaryName 指定该工具生成的二进制文件名（可选）；
		// 若为空则使用默认规则（例如 go install 的默认产物名）
		BinaryName string
	}

	// InstallType 定义了内置工具的安装类型
	InstallType struct {
		Name string
		OS   string
		Arch string
	}
)

// BuiltinTools 内部维护工具的名称和安装路径（既可 go install，也可 clone 构建）
var BuiltinTools = map[string]BuiltinToolsInfo{
	"air":           {Name: "air", URL: "github.com/air-verse/air@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"golangci-lint": {Name: "golangci-lint", URL: "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"buf":           {Name: "buf", URL: "github.com/bufbuild/buf/cmd/buf@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"httpx":         {Name: "httpx", URL: "github.com/projectdiscovery/httpx/cmd/httpx@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"grpcurl":       {Name: "grpcurl", URL: "github.com/fullstorydev/grpcurl/cmd/grpcurl@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"goreleaser":    {Name: "goreleaser", URL: "github.com/goreleaser/goreleaser/v2@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"gojq":          {Name: "gojq", URL: "github.com/itchyny/gojq/cmd/gojq@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"glow":          {Name: "glow", URL: "github.com/charmbracelet/glow/cmd/glow@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"jid":           {Name: "jid", URL: "github.com/simeji/jid/cmd/jid@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"lazygit":       {Name: "lazygit", URL: "github.com/jesseduffield/lazygit@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"dasel":         {Name: "dasel", URL: "github.com/tomwright/dasel/v2/cmd/dasel@master", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"lf":            {Name: "lf", URL: "github.com/gokcehan/lf@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
	"yq":            {Name: "yq", URL: "github.com/mikefarah/yq/v4/cmd/yq@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
}
