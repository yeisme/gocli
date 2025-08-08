// Package fsop provides file system operations.
package fsop

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/yeisme/gocli/pkg/utils/gitignore"
)

// walkSubdirectories 是内部通用实现，支持 ignorePatterns（可为 nil）。
func walkSubdirectories(root string, ignorePatterns []string) ([]string, error) {
	var subdirs []string
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != root {
			if len(ignorePatterns) > 0 && ignorePatterns != nil {
				rel, _ := filepath.Rel(root, path)
				for _, pat := range ignorePatterns {
					if strings.HasPrefix(rel, pat) || strings.HasSuffix(rel, pat) || strings.Contains(rel, pat) {
						return filepath.SkipDir
					}
				}
			}
			subdirs = append(subdirs, path)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return subdirs, nil
}

// walkSubdirectoriesWithGitIgnore 使用 gitignore 包进行更精确的过滤
func walkSubdirectoriesWithGitIgnore(root string, gi *gitignore.GitIgnore) ([]string, error) {
	var subdirs []string
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != root {
			// 获取相对路径用于匹配
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}

			// 检查是否应该被忽略
			if gi.IsIgnored(rel) {
				return filepath.SkipDir
			}

			subdirs = append(subdirs, path)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return subdirs, nil
}

// ListAllSubdirectories lists all subdirectories in the given path, recursively.
// It does not include the root path itself in the returned list.
func ListAllSubdirectories(root string) ([]string, error) {
	return walkSubdirectories(root, nil)
}

// ListAllSubdirectoriesWithIgnore lists all subdirectories, skipping those matching ignorePatterns.
func ListAllSubdirectoriesWithIgnore(root string, ignorePatterns []string) ([]string, error) {
	return walkSubdirectories(root, ignorePatterns)
}

// ListAllSubdirectoriesWithGitIgnore lists all subdirectories, respecting .gitignore rules.
// It loads the .gitignore file from the root directory and applies the rules.
func ListAllSubdirectoriesWithGitIgnore(root string) ([]string, error) {
	gi, err := gitignore.LoadGitIgnoreFromDir(root)
	if err != nil {
		// If we can't load .gitignore, fall back to listing all subdirectories
		return ListAllSubdirectories(root)
	}

	return walkSubdirectoriesWithGitIgnore(root, gi)
}

// ListAllSubdirectoriesWithCustomGitIgnore lists all subdirectories using a custom GitIgnore instance.
func ListAllSubdirectoriesWithCustomGitIgnore(root string, gi *gitignore.GitIgnore) ([]string, error) {
	return walkSubdirectoriesWithGitIgnore(root, gi)
}
