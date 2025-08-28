package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// CloneBuildOptions 描述 clone 后的构建行为
type CloneBuildOptions struct {
	CloneURL          string
	InstallDir        string
	Env               []string
	Verbose           bool
	RecurseSubmodules bool
	BuildMethod       string // "make" | "goreleaser" | 其他通过注册表拓展
	MakeTarget        string
	BuildArgs         []string
	WorkDir           string
	GoreleaserConfig  string
	BinDirs           []string
	BinaryName        string
	Force             bool // 强制模型，如果目标目录已存在则覆盖，否则就复用
}

// CloneAndBuildInstall 克隆仓库并按指定构建方式构建，然后从 bin 目录收集产物
func CloneAndBuildInstall(o CloneBuildOptions) (string, error) {
	// 解析 clone 输入（不删除已有目录；是否删除由复用逻辑控制）
	repoURL, resolvedRef, displayRef, absBase, repoDir, env2, err := resolveCloneInputs(o.CloneURL, o.InstallDir, o.Env, o.Force)
	if err != nil {
		return "", err
	}
	// 克隆检出
	outClone, err := gitCloneAndCheckoutWithOpts(repoURL, repoDir, absBase, resolvedRef, o.RecurseSubmodules)
	if err != nil {
		return outClone, err
	}
	// 计算构建目录
	buildDir := repoDir
	if strings.TrimSpace(o.WorkDir) != "" {
		wd := expandPath(o.WorkDir)
		if !filepath.IsAbs(wd) {
			buildDir = filepath.Join(repoDir, wd)
		} else {
			buildDir = wd
		}
	}

	// 选择构建器：默认 make
	method := strings.ToLower(strings.TrimSpace(o.BuildMethod))
	if method == "" {
		method = "make"
	}
	runner, ok := GetBuildRunner(method)
	if !ok {
		return "", fmt.Errorf("unsupported build method: %s", o.BuildMethod)
	}

	ctx := BuildContext{
		RepoDir:     repoDir,
		BuildDir:    buildDir,
		AbsBase:     absBase,
		RepoURL:     repoURL,
		ResolvedRef: resolvedRef,
		DisplayRef:  displayRef,
		Env:         env2,
		Verbose:     o.Verbose,
	}
	params := BuildParams{
		MakeTarget:       o.MakeTarget,
		GoreleaserConfig: o.GoreleaserConfig,
		BuildArgs:        o.BuildArgs,
	}

	out, err := runner.Build(ctx, params)
	if err != nil {
		return out, err
	}

	// 复制产物：优先使用用户指定的 BinDirs，否则用 runner 默认目录
	binDirs := o.BinDirs
	if len(binDirs) == 0 {
		binDirs = runner.DefaultBinDirs()
	}
	if len(binDirs) > 0 {
		copyOut, err := collectAndCopyBins(binDirs, buildDir, env2, absBase, o.Verbose, o.BinaryName)
		out += copyOut
		if err != nil {
			return out, err
		}
	}
	return out, nil
}
