package models

// InitOptions 初始化的配置选项
type InitOptions struct {
	// Go golang 项目的初始化配置选项
	Go GoInitOptions

	// GitInit 是否自动运行 `git init`（默认 true）
	GitInit bool `mapstructure:"git_init" jsonschema:"title=GitInit,description=Run 'git init' automatically"`

	// License 要添加的许可证标识（例如 "MIT", "Apache-2.0"）；为空表示不生成 LICENSE 文件
	License string `mapstructure:"license" jsonschema:"title=License,description=SPDX license identifier (e.g. MIT,Apache-2.0),nullable"`

	// Dir 要在其中初始化项目的工作目录（可以是相对或绝对路径）
	// 为空表示使用当前工作目录
	Dir string `mapstructure:"dir" jsonschema:"title=Dir,description=Target directory to initialize project in (relative or absolute),nullable"`

	// GoCLIInit 是否初始化 GoCLI Config
	GoCLIInit bool `mapstructure:"gocli" jsonschema:"title=GoCLIInit,description=Create gocli configuration file"`

	// GoreleaserInit 是否初始化 goreleaser 配置文件
	GoreleaserInit bool `mapstructure:"goreleaser" jsonschema:"title=GoreleaserInit,description=Generate goreleaser config"`

	// GoTaskInit 是否初始化 go-task 配置文件
	GoTaskInit bool `mapstructure:"go_task" jsonschema:"title=GoTaskInit,description=Generate go-task Taskfile"`

	// DockerInit 是否初始化 Docker 配置文件
	DockerInit bool `mapstructure:"docker" jsonschema:"title=DockerInit,description=Generate Docker related files"`

	// MakefileInit 是否初始化 Makefile
	MakefileInit bool `mapstructure:"makefile" jsonschema:"title=MakefileInit,description=Generate Makefile"`

	// Author 作者
	Author string `mapstructure:"author" jsonschema:"title=Author,description=Project author name,nullable"`

	// Email 作者邮箱
	Email string `mapstructure:"email" jsonschema:"title=Email,description=Project author email,format=email,nullable"`
}

// GoInitOptions golang 项目的初始化配置选项
type GoInitOptions struct {
	// TemplateType 模板类型
	// 短名 builtinGoTemplates 中的键（例如 "basic", "empty" etc.）
	// 或直接指向一个包含 go.mod 的目录/仓库
	// 默认: "empty"
	Templates map[string]GoFileTemplate `json:"templates"`
}

// GoFileTemplate 用于指定 Go 模板文件的配置
type GoFileTemplate struct {
	// Path 用于指定模板文件的路径
	// 例如 `stdweb` 对应 templates/go/stdweb
	Path string `json:"path"`

	// Type embed/http(s)/git/file_system
	Type string `json:"type"`

	// Language 模板所属编程语言（可选，默认 go）
	Language string `json:"language"`
}
