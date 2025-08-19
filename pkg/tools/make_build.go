package tools

import (
	"fmt"
	"strings"
)

// MakeRunner 实现 BuildRunner
type MakeRunner struct{}

// Name 实现 BuildRunner 接口
func (MakeRunner) Name() string { return "make" }

// DefaultBinDirs 实现 BuildRunner 接口
func (MakeRunner) DefaultBinDirs() []string { return nil }

// Build 实现 BuildRunner 接口
func (MakeRunner) Build(ctx BuildContext, params BuildParams) (string, error) {
	var (
		out string
		err error
	)
	if params.MakeTarget != "" {
		out, err = NewExecutor("make", params.MakeTarget).WithDir(ctx.BuildDir).WithEnv(ctx.Env...).CombinedOutput()
		if err != nil {
			return out, fmt.Errorf("make %s failed: %w", params.MakeTarget, err)
		}
		if ctx.Verbose {
			b := &strings.Builder{}
			fmt.Fprintf(b, "\n[make] target: %s\n", params.MakeTarget)
			fmt.Fprintf(b, "[make] base dir: %s\n", ctx.AbsBase)
			fmt.Fprintf(b, "[make] repo dir: %s\n", ctx.RepoDir)
			fmt.Fprintf(b, "[make] build dir: %s\n", ctx.BuildDir)
			fmt.Fprintf(b, "[make] repo: %s\n", ctx.RepoURL)
			if ctx.ResolvedRef != "" {
				if ctx.DisplayRef == "latest" && ctx.ResolvedRef != "" {
					fmt.Fprintf(b, "[make] checkout: latest -> %s\n", ctx.ResolvedRef)
				} else {
					fmt.Fprintf(b, "[make] checkout: %s\n", ctx.ResolvedRef)
				}
			}
			if gobin := envLookup(ctx.Env, "GOBIN"); gobin != "" {
				fmt.Fprintf(b, "[make] env.GOBIN: %s\n", gobin)
			}
			out += "\n" + b.String()
		}
		return out, nil
	}

	out, err = NewExecutor("make").WithDir(ctx.BuildDir).WithEnv(ctx.Env...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("make failed: %w", err)
	}
	if ctx.Verbose {
		b := &strings.Builder{}
		fmt.Fprintf(b, "\n[make] target: <default>\n")
		fmt.Fprintf(b, "[make] base dir: %s\n", ctx.AbsBase)
		fmt.Fprintf(b, "[make] repo dir: %s\n", ctx.RepoDir)
		fmt.Fprintf(b, "[make] build dir: %s\n", ctx.BuildDir)
		fmt.Fprintf(b, "[make] repo: %s\n", ctx.RepoURL)
		if ctx.ResolvedRef != "" {
			if ctx.DisplayRef == "latest" && ctx.ResolvedRef != "" {
				fmt.Fprintf(b, "[make] checkout: latest -> %s\n", ctx.ResolvedRef)
			} else {
				fmt.Fprintf(b, "[make] checkout: %s\n", ctx.ResolvedRef)
			}
		}
		if gobin := envLookup(ctx.Env, "GOBIN"); gobin != "" {
			fmt.Fprintf(b, "[make] env.GOBIN: %s\n", gobin)
		}
		out += "\n" + b.String()
	}
	return out, nil
}

func init() { RegisterBuildRunner(MakeRunner{}) }
