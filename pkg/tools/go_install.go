package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// goInstallWithEnv 支持传入额外环境变量（如 GOBIN）
func goInstallWithEnv(spec string, env []string, verbose bool, buildArgs []string) (string, error) {
	// 正确传递参数，避免将 "install -v" 作为一个整体参数
	args := []string{"install"}
	if verbose {
		args = append(args, "-v")
	}
	// 追加构建参数（例如 -trimpath、-ldflags=...、-gcflags=...）
	if len(buildArgs) > 0 {
		args = append(args, buildArgs...)
	}
	args = append(args, spec)
	ex := NewExecutor("go", args...)
	if len(env) > 0 {
		ex = ex.WithEnv(env...)
	}
	out, err := ex.CombinedOutput()
	if err != nil {
		return out, err
	}
	return out, nil
}

// InstallGoTool 安装 Go 工具
//   - spec: go install 的目标（模块路径或本地路径），可带 @version
//   - installDir: 若不为空，作为目标安装目录（将通过 GOBIN 传入）；支持 $ENV 与 ~ 展开
//   - env: 额外环境变量（如 CGO_ENABLED=1）
//
// 返回：命令输出、最终绝对安装目录（若设置了 installDir）、错误
func InstallGoTool(spec, installDir string, env []string, verbose bool, buildArgs []string) (string, string, error) {
	finalDir := ""
	env2 := append([]string{}, env...)

	// 规范化 spec：若非本地路径且未显式指定版本，则追加 @latest
	spec = ensureVersionSuffix(spec)
	if installDir != "" {
		p := expandPath(installDir)
		if err := os.MkdirAll(p, 0o755); err != nil {
			return "", "", fmt.Errorf("create install dir %s failed: %w", p, err)
		}
		abs, _ := filepath.Abs(p)
		finalDir = abs
		env2 = append(env2, fmt.Sprintf("GOBIN=%s", abs))
	}
	out, err := goInstallWithEnv(spec, env2, verbose, buildArgs)
	if err != nil {
		return out, finalDir, err
	}
	return out, finalDir, nil
}
