package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// goInstallWithEnv 支持传入额外环境变量（如 GOBIN）

// InstallOptions 用于统一描述工具安装的参数
type InstallOptions struct {
	// go install 专用：模块/本地路径规范，如 github.com/owner/repo/cmd/foo@vX 或 ./path
	Spec string

	// clone+make 专用：仓库 URL，可带 #ref
	CloneURL   string
	MakeTarget string

	// 构建方式（在 clone 模式下有效）："make"(默认)、"goreleaser"、后续可扩展
	BuildMethod string
	// 传递给构建器的附加参数，例如 goreleaser 的 flags 或 make 的额外参数
	BuildArgs []string
	// 在仓库中的子目录进行构建（为空则使用仓库根目录）
	WorkDir string
	// goreleaser 可选配置文件路径（相对于仓库根或 WorkDir）
	GoreleaserConfig string

	// 指定安装目录，等价于设置 GOBIN；若为空，沿用默认规则
	Path string

	// 额外构建/运行环境变量，如 CGO_ENABLED=1
	Env []string

	// 当使用 clone+make 时，从这些目录收集可执行产物并复制到安装目录（可多次传入，或使用平台路径分隔符分隔）
	BinDirs []string

	// 是否 verbose
	Verbose bool
	// 预设构建模式
	ReleaseBuild bool
	DebugBuild   bool

	// BinaryName 指定生成的二进制文件名（可选）
	BinaryName string
	// Clone: 是否递归克隆子模块
	RecurseSubmodules bool

	// Force: 强制模型，如果目标目录已存在则覆盖，否则就复用
	Force bool
}

// InstallResult 统一返回值
type InstallResult struct {
	// 原始命令输出（可能为多行）
	Output string
	// 明确的安装目录（当设置 Path 或 go install 传入 GOBIN 时）
	InstallDir string
	// 根据环境与 go env 推断的安装目录（即便 InstallDir 为空也会提供）
	ProbableInstallDir string
	// 执行模式：go_install 或 clone_make
	Mode string
}

// InstallTool 统一入口：根据是否传入 CloneURL 决定使用 go install 或 clone+make
func InstallTool(opts InstallOptions) (InstallResult, error) {
	res := InstallResult{}
	verbose := opts.Verbose

	// 预处理 env 与 Path -> GOBIN
	env := append([]string{}, opts.Env...)
	finalDir := ""
	if opts.Path != "" {
		p := expandPath(opts.Path)
		if err := os.MkdirAll(p, 0o755); err != nil {
			return res, fmt.Errorf("create install dir %s failed: %w", p, err)
		}
		abs, _ := filepath.Abs(p)
		finalDir = abs
		env = append(env, fmt.Sprintf("GOBIN=%s", abs))
	}

	if opts.CloneURL != "" {
		// 将 Release/Debug 预设尽力通过 GOFLAGS 传递给 make/go build/goreleaser
		var goflags []string
		if opts.ReleaseBuild {
			goflags = append(goflags, "-trimpath", "-ldflags=-s -w")
		}
		if opts.DebugBuild {
			// 注意：GOFLAGS 无法很好地表达带空格的值，这里为最佳努力；部分构建脚本可能不生效
			goflags = append(goflags, "-gcflags=all=-N -l")
		}
		if len(goflags) > 0 {
			env = append(env, "GOFLAGS="+strings.Join(goflags, " "))
		}

		// 由构建器自身提供默认 BinDirs；若用户显式传入则优先生效
		binDirs := append([]string{}, opts.BinDirs...)
		method := strings.ToLower(strings.TrimSpace(opts.BuildMethod))

		out, err := CloneAndBuildInstall(CloneBuildOptions{
			CloneURL:          opts.CloneURL,
			InstallDir:        opts.Path,
			Env:               env,
			Verbose:           verbose,
			RecurseSubmodules: opts.RecurseSubmodules,
			BuildMethod:       method,
			MakeTarget:        opts.MakeTarget,
			BuildArgs:         opts.BuildArgs,
			WorkDir:           opts.WorkDir,
			GoreleaserConfig:  opts.GoreleaserConfig,
			BinDirs:           binDirs,
			BinaryName:        opts.BinaryName,
			Force:             opts.Force,
		})
		res.Output = out
		res.Mode = "clone_build"
		res.InstallDir = finalDir
		// 推断 probable 安装目录
		if finalDir != "" {
			res.ProbableInstallDir = finalDir
		} else {
			p := expandPath(viper.GetString("tools.path"))
			if p == "" {
				if home, e := os.UserHomeDir(); e == nil {
					p = filepath.Join(home, ".gocli", "tools")
				}
			}
			if p != "" {
				if abs, _ := filepath.Abs(p); abs != "" {
					res.ProbableInstallDir = abs
				} else {
					res.ProbableInstallDir = p
				}
			}
		}
		return res, err
	}

	// go install 路径
	var buildArgs []string
	if opts.ReleaseBuild {
		buildArgs = append(buildArgs, "-trimpath", "-ldflags=-s -w")
	}
	if opts.DebugBuild {
		buildArgs = append(buildArgs, "-gcflags=all=-N -l")
	}
	// 在 go install 前快照目标安装目录（若可确定），以便安装后重命名
	var preSnap map[string]time.Time
	var targetDir string
	if finalDir != "" {
		targetDir = finalDir
	} else {
		// 尝试从 go env 推断 GOBIN（为空则回退 GOPATH/bin）
		targetDir = DetermineGoBinDir()
	}
	if targetDir != "" && opts.BinaryName != "" {
		preSnap = SnapshotExecutables(targetDir)
	}

	out, dir, err := InstallGoTool(opts.Spec, opts.Path, env, verbose, buildArgs)
	res.Output = out
	res.Mode = "go_install"
	res.InstallDir = dir
	if dir != "" {
		res.ProbableInstallDir = dir
	}
	// 安装成功并要求自定义二进制名时，尝试重命名
	if err == nil && opts.BinaryName != "" {
		// 优先使用明确安装目录
		renameDir := dir
		if renameDir == "" {
			renameDir = targetDir
		}
		if renameDir != "" {
			if e := RenameInstalledBinary(renameDir, preSnap, opts.BinaryName, verbose); e != nil {
				// 不阻断主流程，仅附加到输出中
				res.Output += "\n[rename] warning: " + e.Error()
			}
		}
	}
	return res, err
}
