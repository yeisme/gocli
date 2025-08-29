// Package newproject 新建项目工具模块
package newproject

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/yeisme/gocli/pkg/tools"
	"github.com/yeisme/gocli/pkg/utils/executor"
)

//go:embed templates/*
var embedTemplateFS embed.FS

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
func (o *InitOptions) ExecConfigInit(args []string) ([]string, error) {
	argsPath, err := NormalizeGoProjectName(args)
	if err != nil {
		return nil, err
	}

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
		str, err := o.execLicenseInit(argsPath)
		initList.AddOutput(str)
		initList.AddError(err)
	}

	if len(initList.ErrList) > 0 {
		return initList.BufList, &initList
	}
	return initList.BufList, nil
}

func (o *InitOptions) execGitInit() (string, error) {
	return executor.NewExecutor("git", "init").WithDir(o.Dir).Output()
}

func (o *InitOptions) execGoTaskInit() (string, error) {
	out, err := executor.NewExecutor("task", "--init").WithDir(o.Dir).Output()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(o.Dir + "/Taskfile.yml"); os.IsNotExist(err) {
		f, err := os.Open(o.Dir + "/Taskfile.yml")
		if err != nil {
			return "", err
		}
		defer func() {
			if closeErr := f.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close Taskfile.yml: %v\n", closeErr)
			}
		}()

		// TODO 根据项目语言类型生成不同的 Taskfile
	}
	return out, nil
}

func (o *InitOptions) execGoCLIInit() (string, error) {
	return executor.NewExecutor("gocli", "config", "init").WithDir(o.Dir).Output()
}

func (o *InitOptions) execGoreleaserInit() (string, error) {
	return executor.NewExecutor("goreleaser", "init").WithDir(o.Dir).Output()
}

func (o *InitOptions) execDockerInit() (string, error) {
	return executor.NewExecutor("docker", "init").WithDir(o.Dir).Output()
}

func (o *InitOptions) execMakefileInit() (string, error) {
	if o.Dir != "" {
		if err := os.Chdir(o.Dir); err != nil {
			return "", err
		}
	}
	f, err := os.Create("Makefile")
	if err != nil {
		return "", err
	}
	defer f.Close()

	// TODO 根据项目语言类型生成不同的 Makefile

	return "", nil
}

func (o *InitOptions) execLicenseInit(argsPath string) (string, error) {
	args := []string{}
	p, err := tools.TestExists("license")
	if err != nil {
		return "", err
	}
	if o.Author != "" {
		args = append(args, "-n", o.Author)
	}
	if argsPath != "" {
		args = append(args, "-p", argsPath)
	}
	args = append(args, "-o", "LICENSE", o.License)

	return executor.NewExecutor(p, args...).WithDir(o.Dir).Output()
}
