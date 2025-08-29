package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yeisme/gocli/pkg/utils/executor"
)

// runGoreleaserWithContext 执行 goreleaser build，并在 verbose 模式下打印上下文
func runGoreleaserWithContext(dir string, env []string, verbose bool, absBase, repoURL, resolvedRef, displayRef, config string, extraArgs []string) (string, error) {
	args := []string{"build", "--clean", "--snapshot", "--single-target"}
	if strings.TrimSpace(config) != "" {
		args = append(args, "--config", config)
	}
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}
	out, err := executor.NewExecutor("goreleaser", args...).WithDir(dir).WithEnv(env...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("goreleaser build failed: %w", err)
	}
	if verbose {
		b := &strings.Builder{}
		fmt.Fprintf(b, "\n[goreleaser] dir: %s\n", dir)
		fmt.Fprintf(b, "[goreleaser] base dir: %s\n", absBase)
		fmt.Fprintf(b, "[goreleaser] repo: %s\n", repoURL)
		if resolvedRef != "" {
			if displayRef == "latest" && resolvedRef != "" {
				fmt.Fprintf(b, "[goreleaser] checkout: latest -> %s\n", resolvedRef)
			} else {
				fmt.Fprintf(b, "[goreleaser] checkout: %s\n", resolvedRef)
			}
		}
		if gobin := envLookup(env, "GOBIN"); gobin != "" {
			fmt.Fprintf(b, "[goreleaser] env.GOBIN: %s\n", gobin)
		}
		out += "\n" + b.String()
	}
	return out, nil
}

// GoreleaserRunner 实现 BuildRunner
type GoreleaserRunner struct{}

// Name 实现 BuildRunner 接口
func (GoreleaserRunner) Name() string { return "goreleaser" }

// DefaultBinDirs 实现 BuildRunner 接口
func (GoreleaserRunner) DefaultBinDirs() []string { return []string{"dist"} }

// Build 实现 BuildRunner 接口
func (GoreleaserRunner) Build(ctx BuildContext, params BuildParams) (string, error) {
	// 智能解析 goreleaser 配置文件：优先 BuildDir，再回退 RepoDir；支持 .yaml/.yml 自动发现
	resolvedCfg := resolveGoreleaserConfig(ctx.BuildDir, ctx.RepoDir, params.GoreleaserConfig)
	return runGoreleaserWithContext(ctx.BuildDir, ctx.Env, ctx.Verbose, ctx.AbsBase, ctx.RepoURL, ctx.ResolvedRef, ctx.DisplayRef, resolvedCfg, params.BuildArgs)
}

func init() { RegisterBuildRunner(GoreleaserRunner{}) }

// resolveGoreleaserConfig 根据传入值与常见命名规则解析 goreleaser 配置文件路径；返回空字符串表示不显式传递
func resolveGoreleaserConfig(buildDir, _, cfg string) string {
	tryFiles := func(dir string, names ...string) string {
		for _, n := range names {
			if n == "" {
				continue
			}
			p := n
			if !filepath.IsAbs(p) {
				p = filepath.Join(dir, n)
			}
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				if abs, _ := filepath.Abs(p); abs != "" {
					return abs
				}
				return p
			}
		}
		return ""
	}

	// 1) 若用户显式提供 cfg，先在 BuildDir 检查该相对路径，再回退 RepoDir，再尝试将其视为绝对路径
	if strings.TrimSpace(cfg) != "" {
		if p := tryFiles(buildDir, cfg); p != "" {
			return p
		}
		return cfg
	}

	// 2) 未提供时，自动发现常见命名
	if p := tryFiles(buildDir, ".goreleaser.yaml", ".goreleaser.yml"); p != "" {
		return p
	}
	return ""
}
