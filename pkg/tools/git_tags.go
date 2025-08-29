package tools

import (
	"fmt"
	"strings"

	"github.com/yeisme/gocli/pkg/utils/executor"
	"golang.org/x/mod/semver"
)

// resolveLatestGitTag 使用 git ls-remote --tags 列出所有 tag，选择最新的语义化版本
// 优先返回稳定版本（无预发布后缀），若不存在稳定版本，则返回最高的预发布版本
func resolveLatestGitTag(repoURL string) (string, error) {
	out, err := executor.NewExecutor("git", "--no-pager", "ls-remote", "--tags", repoURL).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git ls-remote failed: %w", err)
	}

	// 解析 tag 列表，去重 ^{}
	seen := make(map[string]struct{})
	tags := make([]string, 0)
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
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

	// 为比较使用规范化版本，同时保留原始 tag 以便 checkout 使用
	type cand struct{ norm, orig string }
	var bestStable, bestAny *cand
	for _, t := range tags {
		var norm string
		switch {
		case semver.IsValid(t):
			norm = t
		case semver.IsValid("v" + t):
			norm = "v" + t
		default:
			continue
		}
		if bestAny == nil || semver.Compare(norm, bestAny.norm) > 0 {
			bestAny = &cand{norm: norm, orig: t}
		}
		if semver.Prerelease(norm) == "" { // 稳定版
			if bestStable == nil || semver.Compare(norm, bestStable.norm) > 0 {
				bestStable = &cand{norm: norm, orig: t}
			}
		}
	}

	if bestStable != nil {
		return bestStable.orig, nil
	}
	if bestAny != nil {
		return bestAny.orig, nil
	}
	return "", fmt.Errorf("no semver-like tags found in repo: %s", repoURL)
}
