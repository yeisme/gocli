package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// TestExists 检查指定的工具是否存在于系统的 PATH 中；
// 若不存在且为内置工具，则尝试按其 InstallType 自动安装。
func TestExists(tool string) (string, error) {
	// 1) 先查 PATH
	if p, err := exec.LookPath(tool); err == nil {
		return p, nil
	}

	bi, ok := BuiltinTools[tool]
	if !ok {
		return "", fmt.Errorf("tool '%s' not found in PATH and is not a builtin tool", tool)
	}

	// 2) 根据内置定义尝试安装
	it := strings.ToLower(strings.TrimSpace(bi.InstallType.Name))
	switch it {
	case "go", "golang":
		// 使用统一的安装入口，默认走 release 构建以获得精简产物
		res, err := InstallTool(InstallOptions{
			Spec:         ensureVersionSuffix(bi.URL),
			ReleaseBuild: true,
		})
		if err != nil {
			// 包含输出信息，便于排查
			if res.Output != "" {
				return "", fmt.Errorf("install builtin tool '%s' failed: %w\n%s", tool, err, res.Output)
			}
			return "", fmt.Errorf("install builtin tool '%s' failed: %w", tool, err)
		}
		// 安装成功后再走 PATH 检查
		if p, lpErr := exec.LookPath(tool); lpErr == nil {
			return p, nil
		}
		// 若 PATH 未包含安装目录，尝试在可能的安装位置直接解析
		if p := tryResolveInstalledPath(tool, res.InstallDir, res.ProbableInstallDir); p != "" {
			return p, nil
		}
		return "", fmt.Errorf("tool '%s' was installed, but not found in PATH. Please add the install dir to PATH (e.g., GOPATH/bin, GOBIN, or tools.path)", tool)

	case "":
		// 未声明类型，默认按 go 处理
		res, err := InstallTool(InstallOptions{Spec: ensureVersionSuffix(bi.URL), ReleaseBuild: true})
		if err != nil {
			if res.Output != "" {
				return "", fmt.Errorf("install builtin tool '%s' failed: %w\n%s", tool, err, res.Output)
			}
			return "", fmt.Errorf("install builtin tool '%s' failed: %w", tool, err)
		}
		if p, lpErr := exec.LookPath(tool); lpErr == nil {
			return p, nil
		}
		if p := tryResolveInstalledPath(tool, res.InstallDir, res.ProbableInstallDir); p != "" {
			return p, nil
		}
		return "", fmt.Errorf("tool '%s' was installed, but not found in PATH. Please add the install dir to PATH (e.g., GOPATH/bin, GOBIN, or tools.path)", tool)

	default:
		return "", fmt.Errorf("unsupported builtin install type for '%s': %s", tool, bi.InstallType.Name)
	}
}

// tryResolveInstalledPath 尝试在已知安装目录中直接定位二进制
func tryResolveInstalledPath(tool string, dirs ...string) string {
	exe := tool
	if runtime.GOOS == "windows" {
		exe = tool + ".exe"
	}
	for _, d := range dirs {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		p := filepath.Join(d, exe)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	// 尝试 GOPATH/bin 与环境变量 GOBIN
	if p := os.Getenv("GOBIN"); p != "" {
		if fi, err := os.Stat(filepath.Join(p, exe)); err == nil && !fi.IsDir() {
			return filepath.Join(p, exe)
		}
	}
	// 解析 go env GOPATH（可能为多路径）
	if out, err := NewExecutor("go", "env", "GOPATH").Output(); err == nil {
		for gp := range strings.SplitSeq(strings.TrimSpace(out), string(os.PathListSeparator)) {
			gp = strings.TrimSpace(gp)
			if gp == "" {
				continue
			}
			cand := filepath.Join(gp, "bin", exe)
			if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
				return cand
			}
		}
	}
	return ""
}
