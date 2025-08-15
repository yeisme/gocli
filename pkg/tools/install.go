package tools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"golang.org/x/mod/semver"
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

// CloneAndMakeInstall 从 Git 仓库克隆源码并执行 make（可选目标）
//   - cloneURL: 支持 "url#ref" 语法指定分支/标签/提交
//   - makeTarget: 为空则执行 make 默认目标
//   - env: 传递给 make 的环境变量
func CloneAndMakeInstall(cloneURL, installDir, makeTarget string, env []string, verbose bool, binDirs []string) (string, error) {
	// 1) 解析输入并准备目录/环境
	repoURL, resolvedRef, displayRef, absBase, repoDir, env2, err := resolveCloneInputs(cloneURL, installDir, env)
	if err != nil {
		return "", err
	}

	// 2) 克隆并检出
	if out, err := gitCloneAndCheckout(repoURL, repoDir, absBase, resolvedRef); err != nil {
		return out, err
	}

	// 3) 执行 make（带上下文信息）
	out, err := runMakeWithContext(repoDir, makeTarget, env2, verbose, absBase, repoURL, resolvedRef, displayRef)
	if err != nil {
		return out, err
	}

	// 4) 可选复制产物
	if len(binDirs) > 0 {
		copyOut, err := collectAndCopyBins(binDirs, repoDir, env2, absBase, verbose)
		out += copyOut
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// resolveCloneInputs 解析 clone 规格、latest 标签、基础目录与目标仓库目录，并补全 GOBIN
func resolveCloneInputs(cloneURL, installDir string, env []string) (repoURL, resolvedRef, displayRef, absBase, repoDir string, env2 []string, err error) {
	env2 = append([]string{}, env...)
	repoURL, ref := splitRepoAndRef(cloneURL)

	// 未指定 ref -> 解析 latest；目录名显示 latest，检出具体 tag
	resolvedRef = ref
	displayRef = ref
	if strings.TrimSpace(ref) == "" {
		var latest string
		latest, err = resolveLatestGitTag(repoURL)
		if err != nil {
			return
		}
		resolvedRef = latest
		displayRef = "latest"
	}

	// 基础目录
	baseDir := installDir
	if strings.TrimSpace(baseDir) == "" {
		baseDir = viper.GetString("tools.path")
	}
	baseDir = expandPath(baseDir)
	if strings.TrimSpace(baseDir) == "" {
		if home, e := os.UserHomeDir(); e == nil {
			baseDir = filepath.Join(home, ".gocli", "tools")
		}
	}
	if baseDir == "" {
		err = fmt.Errorf("cannot resolve tools base dir")
		return
	}
	if e := os.MkdirAll(baseDir, 0o755); e != nil {
		err = fmt.Errorf("create base dir %s failed: %w", baseDir, e)
		return
	}
	absBase, _ = filepath.Abs(baseDir)

	// 确保 GOBIN
	if envLookup(env2, "GOBIN") == "" {
		env2 = append(env2, "GOBIN="+absBase)
	}

	// 目标仓库目录
	repoName := sanitizeName(extractRepoName(repoURL))
	repoDir = filepath.Join(absBase, "src", repoName)
	if displayRef != "" {
		repoDir = filepath.Join(absBase, "src", repoName+"@"+sanitizeName(displayRef))
	}
	if e := os.MkdirAll(filepath.Dir(repoDir), 0o755); e != nil {
		err = fmt.Errorf("prepare repo parent dir failed: %w", e)
		return
	}
	// 清理旧目录
	_ = os.RemoveAll(repoDir)
	return
}

// gitCloneAndCheckout 克隆仓库并检出指定 ref（若非空）
func gitCloneAndCheckout(repoURL, repoDir, absBase, resolvedRef string) (string, error) {
	if out, err := NewExecutor("git", "clone", repoURL, repoDir).WithDir(absBase).CombinedOutput(); err != nil {
		return out, fmt.Errorf("git clone failed: %w", err)
	}
	if strings.TrimSpace(resolvedRef) == "" {
		return "", nil
	}
	if out, err := NewExecutor("git", "checkout", resolvedRef).WithDir(repoDir).CombinedOutput(); err != nil {
		return out, fmt.Errorf("git checkout %s failed: %w", resolvedRef, err)
	}
	return "", nil
}

// runMakeWithContext 执行 make 并在 verbose 模式下附加上下文信息
func runMakeWithContext(repoDir, makeTarget string, env []string, verbose bool, absBase, repoURL, resolvedRef, displayRef string) (string, error) {
	var (
		out string
		err error
	)
	if makeTarget != "" {
		out, err = NewExecutor("make", makeTarget).WithDir(repoDir).WithEnv(env...).CombinedOutput()
		if err != nil {
			return out, fmt.Errorf("make %s failed: %w", makeTarget, err)
		}
		if verbose {
			b := &strings.Builder{}
			fmt.Fprintf(b, "\n[make] target: %s\n", makeTarget)
			fmt.Fprintf(b, "[make] base dir: %s\n", absBase)
			fmt.Fprintf(b, "[make] repo dir: %s\n", repoDir)
			fmt.Fprintf(b, "[make] repo: %s\n", repoURL)
			if resolvedRef != "" {
				if displayRef == "latest" && resolvedRef != "" {
					fmt.Fprintf(b, "[make] checkout: latest -> %s\n", resolvedRef)
				} else {
					fmt.Fprintf(b, "[make] checkout: %s\n", resolvedRef)
				}
			}
			if gobin := envLookup(env, "GOBIN"); gobin != "" {
				fmt.Fprintf(b, "[make] env.GOBIN: %s\n", gobin)
			}
			out += "\n" + b.String()
		}
		return out, nil
	}

	out, err = NewExecutor("make").WithDir(repoDir).WithEnv(env...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("make failed: %w", err)
	}
	if verbose {
		b := &strings.Builder{}
		fmt.Fprintf(b, "\n[make] target: <default>\n")
		fmt.Fprintf(b, "[make] base dir: %s\n", absBase)
		fmt.Fprintf(b, "[make] repo dir: %s\n", repoDir)
		fmt.Fprintf(b, "[make] repo: %s\n", repoURL)
		if resolvedRef != "" {
			if displayRef == "latest" && resolvedRef != "" {
				fmt.Fprintf(b, "[make] checkout: latest -> %s\n", resolvedRef)
			} else {
				fmt.Fprintf(b, "[make] checkout: %s\n", resolvedRef)
			}
		}
		if gobin := envLookup(env, "GOBIN"); gobin != "" {
			fmt.Fprintf(b, "[make] env.GOBIN: %s\n", gobin)
		}
		out += "\n" + b.String()
	}
	return out, nil
}

// collectAndCopyBins 从给定 bin 目录收集可执行文件复制到安装目录，并返回日志追加文本
func collectAndCopyBins(binDirs []string, repoDir string, env []string, absBase string, verbose bool) (string, error) {
	// 目标安装目录：优先 env 中的 GOBIN，否则 absBase
	destDir := envLookup(env, "GOBIN")
	if destDir == "" {
		destDir = absBase
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("prepare install dir %s failed: %w", destDir, err)
	}

	// 归一化 binDirs
	flatDirs := make([]string, 0, len(binDirs))
	for _, d := range binDirs {
		for part := range strings.SplitSeq(d, string(os.PathListSeparator)) {
			part = strings.TrimSpace(part)
			if part != "" {
				flatDirs = append(flatDirs, part)
			}
		}
	}

	b := &strings.Builder{}
	copied := 0
	for _, d := range flatDirs {
		dd := expandPath(d)
		if !filepath.IsAbs(dd) {
			dd = filepath.Join(repoDir, dd)
		}
		entries, err := os.ReadDir(dd)
		if err != nil {
			if verbose {
				fmt.Fprintf(b, "\n[bin] skip dir (not found): %s", dd)
			}
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			srcName := e.Name()
			if !isExecutable(srcName, dd) {
				continue
			}
			srcPath := filepath.Join(dd, srcName)
			dstPath := filepath.Join(destDir, srcName)
			if err := copyFile(srcPath, dstPath); err != nil {
				return b.String(), fmt.Errorf("copy %s -> %s failed: %w", srcPath, dstPath, err)
			}
			copied++
			if verbose {
				fmt.Fprintf(b, "\n[bin] copied: %s -> %s", srcPath, dstPath)
			}
		}
	}
	if verbose {
		fmt.Fprintf(b, "\n[bin] total copied executables: %d", copied)
	}
	return b.String(), nil
}

func expandPath(p string) string {
	if p == "" {
		return p
	}
	// 展开环境变量
	p = os.ExpandEnv(p)
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			// 处理 ~/xxx
			if len(p) >= 2 && (p[1] == '/' || p[1] == '\\') {
				return filepath.Join(home, p[2:])
			}
		}
	}
	return p
}

func splitRepoAndRef(s string) (string, string) {
	idx := strings.LastIndex(s, "#")
	if idx <= 0 || idx == len(s)-1 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}

// ensureVersionSuffix 在满足以下条件时为 spec 追加 @latest：
//   - 没有包含 '@'
//   - 不是本地路径（不是绝对路径；不以 ./ 或 ../ 开头；路径在本地不存在）
func ensureVersionSuffix(spec string) string {
	if spec == "" || strings.Contains(spec, "@") {
		return spec
	}

	// 明显的本地路径：绝对路径、./ 或 ../
	if filepath.IsAbs(spec) || strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") ||
		strings.HasPrefix(spec, ".\\") || strings.HasPrefix(spec, "..\\") {
		return spec
	}

	// 若文件系统上确实存在该路径（目录或文件），视为本地
	if _, err := os.Stat(spec); err == nil {
		return spec
	}

	// 其他情况，当作模块路径，追加 @latest
	return spec + "@latest"
}

// envLookup 在形如 KEY=VAL 的切片中查找 KEY 的值
func envLookup(env []string, key string) string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return strings.TrimPrefix(e, prefix)
		}
	}
	return ""
}

// extractRepoName 尽力从仓库 URL 中提取仓库名（去除 .git 尾缀）
func extractRepoName(u string) string {
	s := u
	// 去掉片段
	if i := strings.LastIndex(s, "#"); i >= 0 {
		s = s[:i]
	}
	// 兼容 ssh/https：取最后一个路径分隔符后的名字
	idx := strings.LastIndexAny(s, "/:")
	if idx >= 0 && idx+1 < len(s) {
		s = s[idx+1:]
	}
	s = strings.TrimSuffix(s, ".git")
	if s == "" {
		s = "repo"
	}
	return s
}

// sanitizeName 清理为文件夹安全名（Windows 也安全）
func sanitizeName(s string) string {
	// 替换常见非法字符
	r := strings.NewReplacer(
		"<", "-",
		">", "-",
		":", "-",
		"\"", "-",
		"|", "-",
		"?", "-",
		"*", "-",
	)
	s = r.Replace(s)
	s = strings.TrimSpace(s)
	if s == "" {
		return "repo"
	}
	return s
}

// copyFile 以 0644 权限复制普通文件（若目标存在则覆盖）
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	// 确保目标目录存在
	if mkErr := os.MkdirAll(filepath.Dir(dst), 0o755); mkErr != nil {
		return mkErr
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(0o644)
}

// resolveLatestGitTag 使用 git ls-remote --tags 列出所有 tag，选择最新的语义化版本
// 优先返回稳定版本（无预发布后缀），若不存在稳定版本，则返回最高的预发布版本
func resolveLatestGitTag(repoURL string) (string, error) {
	out, err := NewExecutor("git", "--no-pager", "ls-remote", "--tags", repoURL).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git ls-remote failed: %w", err)
	}

	// 解析 tag 列表，去重 ^{}
	seen := make(map[string]struct{})
	tags := make([]string, 0)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ref := fields[len(fields)-1]
		const prefix = "refs/tags/"
		if !strings.HasPrefix(ref, prefix) {
			continue
		}
		t := strings.TrimPrefix(ref, prefix)
		t = strings.TrimSuffix(t, "^{}")
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		tags = append(tags, t)
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found in repo: %s", repoURL)
	}

	normalize := func(tag string) (norm string, ok bool) {
		if semver.IsValid(tag) {
			return tag, true
		}
		// 尝试为不带 v 的版本添加前缀
		if semver.IsValid("v" + tag) {
			return "v" + tag, true
		}
		return "", false
	}

	bestStable := ""
	bestAny := ""
	for _, t := range tags {
		v, ok := normalize(t)
		if !ok {
			continue
		}
		// bestAny
		if bestAny == "" || semver.Compare(v, bestAny) > 0 {
			bestAny = v
		}
		// 稳定版（无预发布）
		if semver.Prerelease(v) == "" {
			if bestStable == "" || semver.Compare(v, bestStable) > 0 {
				bestStable = v
			}
		}
	}

	if bestStable != "" {
		return bestStable, nil
	}
	if bestAny != "" {
		return bestAny, nil
	}
	return "", fmt.Errorf("no semver-like tags found in repo: %s", repoURL)
}

// InstallOptions 用于统一描述工具安装的参数
type InstallOptions struct {
	// go install 专用：模块/本地路径规范，如 github.com/owner/repo/cmd/foo@vX 或 ./path
	Spec string

	// clone+make 专用：仓库 URL，可带 #ref
	CloneURL   string
	MakeTarget string

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
		// 将 Release/Debug 预设尽力通过 GOFLAGS 传递给 make/go build
		var goflags []string
		if opts.ReleaseBuild {
			goflags = append(goflags, "-trimpath", "-ldflags=-s -w")
		}
		if opts.DebugBuild {
			// 注意：GOFLAGS 无法很好地表达带空格的值，这里为最佳努力；部分 Makefile 可能不生效
			goflags = append(goflags, "-gcflags=all=-N -l")
		}
		if len(goflags) > 0 {
			env = append(env, "GOFLAGS="+strings.Join(goflags, " "))
		}
		out, err := CloneAndMakeInstall(opts.CloneURL, opts.Path, opts.MakeTarget, env, verbose, opts.BinDirs)
		res.Output = out
		res.Mode = "clone_make"
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
	out, dir, err := InstallGoTool(opts.Spec, opts.Path, env, verbose, buildArgs)
	res.Output = out
	res.Mode = "go_install"
	res.InstallDir = dir
	if dir != "" {
		res.ProbableInstallDir = dir
	}
	return res, err
}
