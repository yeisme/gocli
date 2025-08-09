// Package tools provides utilities and shared functionality for the `tools` command and its subcommands.
// It also includes reusable components that can be utilized by other commands or shared across multiple subcommands.
package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type toolSourceType string

const (
	goPath    toolSourceType = "$GOPATH/bin"
	goCliPath toolSourceType = "$GOCLI_TOOLS_PATH"
)

// ToolInfo 描述一个可用的工具
type ToolInfo struct {
	Name   string         `json:"name"`
	Path   string         `json:"path"`
	Source toolSourceType `json:"source"`
}

// FindTools 搜索可用工具，来源包括：
// - GOPATH/bin 下的可执行文件
// - 用户目录下的 .gocli/tools 下的可执行文件（优先级更高，覆盖同名）
// verbose 目前保留参数，不影响返回结果，预留将来扩展
func FindTools(_ bool, gocliToolsPath string) []ToolInfo {
	// 收集两类目录
	dirs := make([]struct {
		path   string
		source toolSourceType
	}, 0, 4)

	// 1) 从 GOPATH 解析多个路径，并拼接 bin 目录
	for _, gp := range getGoPaths() {
		if gp == "" {
			continue
		}
		dirs = append(dirs, struct {
			path   string
			source toolSourceType
		}{path: joinPath(gp, "bin"), source: goPath})
	}

	// 2) 用户目录下的 .gocli/tools
	if userTools := getUserToolsDir(gocliToolsPath); userTools != "" {
		dirs = append(dirs, struct {
			path   string
			source toolSourceType
		}{path: userTools, source: goCliPath})
	}

	// 先扫描 GOPATH/bin，再用 .gocli/tools 覆盖（保证用户自定义优先）
	resultMap := map[string]ToolInfo{}
	for _, d := range dirs {
		for _, ti := range listExecutablesInDir(d.path, d.source) {
			// 覆盖策略：后出现的覆盖先前的
			resultMap[ti.Name] = ti
		}
	}

	// 转为切片并按名称排序
	out := make([]ToolInfo, 0, len(resultMap))
	for _, v := range resultMap {
		out = append(out, v)
	}
	sortTools(out)
	return out
}

// --- helpers ---

func getGoPaths() []string {
	// 优先环境变量
	if gp := os.Getenv("GOPATH"); gp != "" {
		return splitList(gp)
	}
	// 备用从 `go env GOPATH` 获取
	out, err := NewExecutor("go", "env", "GOPATH").Output()
	if err == nil {
		return splitList(strings.TrimSpace(out))
	}
	return nil
}

func splitList(s string) []string {
	// 按平台路径分隔符切分（Windows 为 ;，Unix 为 :）
	parts := strings.Split(s, string(os.PathListSeparator))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getUserToolsDir(gocliToolsPath string) string {
	if gocliToolsPath != "" {
		return gocliToolsPath
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".gocli", "tools")
}

func joinPath(elem ...string) string {
	return filepath.Join(elem...)
}

func listExecutablesInDir(dir string, source toolSourceType) []ToolInfo {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var tools []ToolInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isExecutable(name, dir) {
			continue
		}
		display := stripExeSuffix(name)
		tools = append(tools, ToolInfo{
			Name:   display,
			Path:   filepath.Join(dir, name),
			Source: source,
		})
	}
	return tools
}

func isExecutable(name, dir string) bool {
	// Windows: 允许常见可执行后缀；非 Windows: 通过权限位判断
	if runtime.GOOS == "windows" {
		lower := strings.ToLower(name)
		return strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".bat") || strings.HasSuffix(lower, ".cmd") || strings.HasSuffix(lower, ".ps1")
	}
	// 非 Windows：检查是否设置了可执行位
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		return false
	}
	mode := info.Mode()
	return mode&0o111 != 0
}

func stripExeSuffix(name string) string {
	if runtime.GOOS == "windows" {
		lower := strings.ToLower(name)
		for _, ext := range []string{".exe", ".bat", ".cmd", ".ps1"} {
			if strings.HasSuffix(lower, ext) {
				return name[:len(name)-len(ext)]
			}
		}
	}
	return name
}

func sortTools(ts []ToolInfo) {
	sort.Slice(ts, func(i, j int) bool {
		if ts[i].Name == ts[j].Name {
			// 稳定次序：用户目录优先
			if ts[i].Source == ts[j].Source {
				return ts[i].Path < ts[j].Path
			}
			return ts[i].Source == goCliPath
		}
		return ts[i].Name < ts[j].Name
	})
}
