package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// collectAndCopyBins 从给定 bin 目录收集可执行文件复制到安装目录，并返回日志追加文本
func collectAndCopyBins(binDirs []string, repoDir string, env []string, absBase string, verbose bool, targetBinary string) (string, error) {
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
	copiedNames := make([]string, 0)
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
			copiedNames = append(copiedNames, e.Name())
			if verbose {
				fmt.Fprintf(b, "\n[bin] copied: %s -> %s", srcPath, dstPath)
			}
		}
	}
	// 如果指定了目标二进制名，且仅复制了一个可执行文件，则在目标目录中重命名
	if targetBinary != "" && copied == 1 && len(copiedNames) == 1 {
		oldName := copiedNames[0]
		newName := targetBinary
		lowerOld := strings.ToLower(oldName)
		lowerNew := strings.ToLower(newName)
		if runtime.GOOS == "windows" {
			// 保留 .exe 等后缀；若新名未带后缀而旧名有 .exe，则补上
			if !strings.HasSuffix(lowerNew, ".exe") && strings.HasSuffix(lowerOld, ".exe") {
				newName += ".exe"
			}
		}
		oldPath := filepath.Join(destDir, oldName)
		newPath := filepath.Join(destDir, newName)
		if oldPath != newPath {
			if err := os.Rename(oldPath, newPath); err != nil {
				return b.String(), fmt.Errorf("rename %s -> %s failed: %w", oldPath, newPath, err)
			}
			if verbose {
				fmt.Fprintf(b, "\n[bin] renamed: %s -> %s", oldPath, newPath)
			}
		}
	}
	if verbose {
		fmt.Fprintf(b, "\n[bin] total copied executables: %d", copied)
	}
	return b.String(), nil
}
