package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

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

// gitCloneAndCheckoutWithOpts 支持递归子模块等可选项
func gitCloneAndCheckoutWithOpts(repoURL, repoDir, absBase, resolvedRef string, recurseSubmodules bool) (string, error) {
	args := []string{"clone"}
	if recurseSubmodules {
		args = append(args, "--recurse-submodules")
	}
	args = append(args, repoURL, repoDir)
	if out, err := NewExecutor("git", args...).WithDir(absBase).CombinedOutput(); err != nil {
		return out, fmt.Errorf("git clone failed: %w", err)
	}
	if strings.TrimSpace(resolvedRef) == "" {
		return "", nil
	}
	if out, err := NewExecutor("git", "checkout", resolvedRef).WithDir(repoDir).CombinedOutput(); err != nil {
		// 回退尝试 tags/<ref>
		if out2, err2 := NewExecutor("git", "checkout", "tags/"+resolvedRef).WithDir(repoDir).CombinedOutput(); err2 == nil {
			return out + "\n" + out2, nil
		}
		return out, fmt.Errorf("git checkout %s failed: %w", resolvedRef, err)
	}
	return "", nil
}
