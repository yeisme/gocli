// Package newproject 新建项目工具模块
package newproject

import (
	"embed"
	"strings"

	"github.com/yeisme/gocli/pkg/tools"
)

//go:embed templates/*
var embedTemplateFS embed.FS

// InitOptions 初始化的配置选项
type InitOptions struct {
	// Go golang 项目的初始化配置选项
	Go GoInitOptions

	// GitInit 是否自动运行 `git init`（默认 true）
	GitInit bool `mapstructure:"git_init"`

	// License 要添加的许可证标识（例如 "MIT", "Apache-2.0"）；为空表示不生成 LICENSE 文件
	License string `mapstructure:"license"`

	// Dir 要在其中初始化项目的工作目录（可以是相对或绝对路径）
	// 为空表示使用当前工作目录
	Dir string `mapstructure:"dir"`

	// GoCLIInit 是否初始化 GoCLI Config
	GoCLIInit bool `mapstructure:"gocli"`

	// GoreleaserInit 是否初始化 goreleaser 配置文件
	GoreleaserInit bool `mapstructure:"goreleaser"`

	// GoTaskInit 是否初始化 go-task 配置文件
	GoTaskInit bool `mapstructure:"go_task"`

	// DockerInit 是否初始化 Docker 配置文件
	DockerInit bool `mapstructure:"docker"`

	// MakefileInit 是否初始化 Makefile
	MakefileInit bool `mapstructure:"makefile"`

	// Author 作者
	Author string `mapstructure:"author"`

	// Email 作者邮箱
	Email string `mapstructure:"email"`
}

// InitList 用于收集初始化过程中产生的输出和错误
type InitList struct {
	// BufList 收集的输出信息
	BufList []string
	// ErrList 收集的错误信息
	ErrList []error
}

// AddError 添加错误信息
func (i *InitList) AddError(err error) {
	if err != nil {
		i.ErrList = append(i.ErrList, err)
	}
}

// Error 返回收集到的错误信息
func (i *InitList) Error() string {
	var sb strings.Builder
	for _, err := range i.ErrList {
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}
	return sb.String()
}

// AddOutput 添加输出信息
func (i *InitList) AddOutput(output string) {
	if strings.TrimSpace(output) != "" {
		i.BufList = append(i.BufList, output)
	}
}

// String 返回收集到的输出信息
func (i *InitList) String() string {
	return strings.Join(i.BufList, "\n")
}

// ExecConfigInit 执行配置初始化
func (o *InitOptions) ExecConfigInit() (string, error) {

	var initList InitList

	if o.GitInit {
		str, err := o.execGitInit()
		initList.AddOutput(str)
		initList.AddError(err)
	}
	if o.GoTaskInit {
		str, err := o.execGoTaskInit()
		initList.AddOutput(str)
		initList.AddError(err)
	}
	if o.GoCLIInit {
		str, err := o.execGoCLIInit()
		initList.AddOutput(str)
		initList.AddError(err)
	}
	if o.GoreleaserInit {
		str, err := o.execGoreleaserInit()
		initList.AddOutput(str)
		initList.AddError(err)
	}
	if o.DockerInit {
		str, err := o.execDockerInit()
		initList.AddOutput(str)
		initList.AddError(err)
	}
	if o.MakefileInit {
		str, err := o.execMakefileInit()
		initList.AddOutput(str)
		initList.AddError(err)
	}
	if o.License != "" {
		str, err := o.execLicenseInit()
		initList.AddOutput(str)
		initList.AddError(err)
	}

	if len(initList.ErrList) > 0 {
		return initList.String(), &initList
	}
	return initList.String(), nil
}

func (o *InitOptions) execGitInit() (string, error) {
	return tools.NewExecutor("git", "init").WithDir(o.Dir).Output()
}

func (o *InitOptions) execGoTaskInit() (string, error) {
	return tools.NewExecutor("task", "--init").WithDir(o.Dir).Output()
}

func (o *InitOptions) execGoCLIInit() (string, error) {
	return tools.NewExecutor("gocli", "config", "init").WithDir(o.Dir).Output()
}

func (o *InitOptions) execGoreleaserInit() (string, error) {
	return tools.NewExecutor("goreleaser", "init").WithDir(o.Dir).Output()
}

func (o *InitOptions) execDockerInit() (string, error) {
	return tools.NewExecutor("docker", "init").WithDir(o.Dir).Output()
}

func (o *InitOptions) execMakefileInit() (string, error) {
	return tools.NewExecutor("makefile", "init").WithDir(o.Dir).Output()
}

func (o *InitOptions) execLicenseInit() (string, error) {
	return tools.NewExecutor("license", "init").WithDir(o.Dir).Output()
}
