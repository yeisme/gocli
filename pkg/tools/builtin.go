package tools

type (
	// BuiltinToolsInfo 定义了内置工具的信息
	BuiltinToolsInfo struct {
		Name        string
		URL         string
		InstallType InstallType
	}

	// InstallType 定义了内置工具的安装类型
	InstallType struct {
		Name string
		OS   string
		Arch string
	}
)

// BuiltinTools 内部维护工具的名称和安装路径
var BuiltinTools = map[string]BuiltinToolsInfo{
	"golangci-lint": {Name: "golangci-lint", URL: "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest", InstallType: InstallType{Name: "Go", OS: "", Arch: ""}},
}
