package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DetermineGoBinDir 尝试通过 `go env` 推断 GOBIN 或 GOPATH/bin
func DetermineGoBinDir() string {
	gobin, _ := NewExecutor("go", "env", "GOBIN").Output()
	gobin = strings.TrimSpace(gobin)
	if gobin != "" {
		if abs, _ := filepath.Abs(expandPath(gobin)); abs != "" {
			return abs
		}
		return expandPath(gobin)
	}
	gopath, _ := NewExecutor("go", "env", "GOPATH").Output()
	gopath = strings.TrimSpace(gopath)
	if gopath == "" {
		return ""
	}
	// 取第一个路径
	first := gopath
	if idx := strings.IndexRune(gopath, os.PathListSeparator); idx >= 0 {
		first = gopath[:idx]
	}
	p := filepath.Join(first, "bin")
	if abs, _ := filepath.Abs(p); abs != "" {
		return abs
	}
	return p
}

// SnapshotExecutables 获取目录下可执行文件的修改时间快照
func SnapshotExecutables(dir string) map[string]time.Time {
	m := make(map[string]time.Time)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return m
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isExecutable(name, dir) {
			continue
		}
		if fi, err := e.Info(); err == nil {
			m[name] = fi.ModTime()
		}
	}
	return m
}

// RenameInstalledBinary 基于安装前后的快照在目录内查找新增/更新的可执行文件，并按需重命名
func RenameInstalledBinary(dir string, pre map[string]time.Time, targetName string, _ bool) error {
	if dir == "" || targetName == "" {
		return nil
	}
	after := SnapshotExecutables(dir)
	// 先找新增文件
	candidates := make([]string, 0)
	for name := range after {
		if _, ok := pre[name]; !ok {
			candidates = append(candidates, name)
		}
	}
	// 若无新增，找修改过的（时间更晚）
	if len(candidates) == 0 {
		for name, mt := range after {
			if pmt, ok := pre[name]; ok {
				if mt.After(pmt) {
					candidates = append(candidates, name)
				}
			}
		}
	}
	if len(candidates) != 1 {
		// 不明确就不改名
		return nil
	}
	oldName := candidates[0]
	newName := targetName
	lowerOld := strings.ToLower(oldName)
	lowerNew := strings.ToLower(newName)
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(lowerNew, ".exe") && strings.HasSuffix(lowerOld, ".exe") {
			newName += ".exe"
		}
	}
	oldPath := filepath.Join(dir, oldName)
	newPath := filepath.Join(dir, newName)
	if oldPath == newPath {
		return nil
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("rename %s -> %s failed: %w", oldPath, newPath, err)
	}
	return nil
}
