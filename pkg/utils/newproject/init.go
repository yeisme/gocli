// Package newproject 新建项目工具模块
package newproject

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/yeisme/gocli/pkg/models"
	"github.com/yeisme/gocli/pkg/tools"
	"github.com/yeisme/gocli/pkg/utils/executor"
)

// InitOptions 用于初始化项目的选项
type InitOptions struct {
	models.InitOptions
}

//go:embed templates/*
var embedTemplateFS embed.FS

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
