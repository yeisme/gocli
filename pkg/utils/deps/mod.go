package deps

import (
	"fmt"
	"os"
	"strings"

	"github.com/yeisme/gocli/pkg/tools"
)

// RunGoModTidy 执行 `go mod tidy`，同步 go.mod 与 go.sum：
//   - 移除未使用的依赖项；
//   - 补全缺失的依赖及校验和；
//   - 按需更新 go.sum
//
// 返回值:
//   - string: 命令的标准输出（可能为空）；
//   - error: 执行失败时返回，错误中已包含底层 stderr 详情
//
// 注意: 不会修改当前进程的工作目录；在调用方的当前工作目录下执行
func RunGoModTidy() (string, error) {
	output, err := tools.NewExecutor("go", "mod", "tidy").Output()
	if err != nil {
		return "", err
	}
	return output, nil
}

// RunGoModVendor 执行 `go mod vendor`，将 go.mod 中声明的依赖复制到 vendor 目录
//
// 返回值与错误约定同 RunGoModTidy
func RunGoModVendor() (string, error) {
	output, err := tools.NewExecutor("go", "mod", "vendor").Output()
	if err != nil {
		return "", err
	}
	return output, nil
}

// RunGoModDownload 执行 `go mod download`，下载并缓存模块依赖（包括 go.sum 校验）
//
// 返回值与错误约定同 RunGoModTidy
func RunGoModDownload() (string, error) {
	output, err := tools.NewExecutor("go", "mod", "download").Output()
	if err != nil {
		return "", err
	}
	return output, nil
}

// RunGoModVerify 执行 `go mod verify`，校验本地模块与其 go.sum 校验和的一致性
//
// 返回值与错误约定同 RunGoModTidy
func RunGoModVerify() (string, error) {
	output, err := tools.NewExecutor("go", "mod", "verify").Output()
	if err != nil {
		return "", err
	}
	return output, nil
}

// RunGoModWhy 执行 `go mod why` 并附带可选标志，用于解释某个包/模块为何被当前模块需要
//
// 选项:
//   - Module: 等价于 `-m`，解释"模块"为何需要（而非具体包）；
//   - Vendor: 等价于 `-vendor`，解释为何需要使用 vendor 中的包；
//
// 参数:
//   - args: 目标包或模块（如 ./..., std, 或具体 import 路径）若为空，默认使用 "./..."
//
// 返回:
//   - string: 标准输出（含依赖链中的说明文本）；
//   - error: 执行错误（包含 stderr 信息）
//
// 示例:
//
//	RunGoModWhy(nil, struct{Module, Vendor bool}{})          // go mod why ./...
//	RunGoModWhy([]string{"std"}, struct{Module, Vendor bool}{Module:true})
//	RunGoModWhy([]string{"github.com/x/y"}, struct{Module, Vendor bool}{})
func RunGoModWhy(args []string, options struct {
	Module bool
	Vendor bool
}) (string, error) {
	base := []string{"mod", "why"}
	if options.Module {
		base = append(base, "-m")
	}
	if options.Vendor {
		base = append(base, "-vendor")
	}
	if len(args) == 0 {
		base = append(base, "./...")
	} else {
		base = append(base, args...)
	}
	output, err := tools.NewExecutor("go", base...).Output()
	if err != nil {
		return "", err
	}
	return output, nil
}

// RunGoModInit 执行 `go mod init [module]`。
//
// 如果 module 为空字符串，则运行 `go mod init` 让 go 命令自动推断模块路径；
// 如果指定了 module，则使用该模块路径初始化。
// 在执行命令前会检查当前目录是否已存在 go.mod，若存在则返回明确的错误信息。
//
// 返回值：命令标准输出，或执行错误（包含底层 stderr 信息）。
// Opts for RunGoModInit
func RunGoModInit(module string, dir string) (string, error) {
	// 检查目标目录下的 go.mod（若 Dir 为空则检查当前工作目录）
	statPath := "go.mod"
	if strings.TrimSpace(dir) != "" {
		statPath = strings.TrimRight(dir, string(os.PathSeparator)) + string(os.PathSeparator) + "go.mod"
	}

	if _, err := os.Stat(statPath); err == nil {
		return "", fmt.Errorf("go.mod already exists in %s", func() string {
			if dir == "" {
				return "current directory"
			}
			return dir
		}())
	} else if !os.IsNotExist(err) {
		return "", err
	}

	args := []string{"mod", "init"}
	if strings.TrimSpace(module) != "" {
		args = append(args, module)
	}

	exec := tools.NewExecutor("go", args...)
	if strings.TrimSpace(dir) != "" {
		exec = exec.WithDir(dir)
	}

	output, err := exec.Output()
	if err != nil {
		return "", err
	}
	return output, nil
}
