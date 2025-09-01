// Package tools provides utilities and shared functionality for the `tools` command and its subcommands.
// It also includes reusable components that can be utilized by other commands or shared across multiple subcommands.
package tools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/executor"
)

type toolSourceType string

const (
	goPath        toolSourceType = "$GOPATH/bin"
	goCliPath     toolSourceType = "$GOCLI_TOOLS_PATH"
	goUserCliPath toolSourceType = "$HOME/.gocli/tools"
)

// ToolInfo 描述一个可用的工具
type ToolInfo struct {
	Name   string         `json:"name"`
	Path   string         `json:"path"`
	Source toolSourceType `json:"source"`
	// 以下字段用于 verbose 展示
	Size    int64     `json:"size,omitempty"`
	ModTime time.Time `json:"modTime,omitzero"`
}

type cached struct {
	once sync.Once
	val  []ToolInfo
}

var toolCachesMap sync.Map // map[string]*cached

// FindTools 搜索可用工具，来源包括：
//   - GOPATH/bin 下的可执行文件
//   - 用户目录下的 .gocli/tools 下的可执行文件（优先级更高，覆盖同名）
//
// verbose 目前保留参数，不影响返回结果，预留将来扩展
// findToolsUnlocked 包含原始的扫描逻辑，不包含任何缓存或并发控制
func findToolsUnlocked(_ bool, gocliToolsPath string) []ToolInfo {
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

	// 2) 用户配置的 .gocli/tools
	if userCfgTools := getUserToolsDir(gocliToolsPath); userCfgTools != "" {
		dirs = append(dirs, struct {
			path   string
			source toolSourceType
		}{path: userCfgTools, source: goCliPath})
	}

	// 3) 用户目录的 .gocli/tools
	if userTools := getUserToolsDir(""); userTools != "" {
		dirs = append(dirs, struct {
			path   string
			source toolSourceType
		}{path: userTools, source: goUserCliPath})
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

// cacheKey 生成缓存 key，目前以 gocliToolsPath 为主键，
// TODO 未来可扩展包含 GOPATH snapshot 等
func cacheKey(gocliToolsPath string) string {
	if gocliToolsPath == "" {
		return "@default"
	}
	return gocliToolsPath
}

// FindTools 在内部使用按 key 的并发安全缓存（每个 key 使用 sync.Once 确保只初始化一次）
func FindTools(verbose bool, gocliToolsPath string) []ToolInfo {
	key := cacheKey(gocliToolsPath)
	v, _ := toolCachesMap.LoadOrStore(key, &cached{})
	c := v.(*cached)
	c.once.Do(func() {
		c.val = findToolsUnlocked(verbose, gocliToolsPath)
	})
	return c.val
}

// ClearToolsCache 清空缓存（可在工具安装/卸载或用户显式请求刷新时调用）
func ClearToolsCache() {
	toolCachesMap.Range(func(k, _ any) bool {
		toolCachesMap.Delete(k)
		return true
	})
}

// --- helpers ---

func getGoPaths() []string {
	// 优先环境变量
	if gp := os.Getenv("GOPATH"); gp != "" {
		return splitList(gp)
	}
	// 备用从 `go env GOPATH` 获取
	out, err := executor.NewExecutor("go", "env", "GOPATH").Output()
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
		// 读取文件信息以获取大小与修改时间
		fi, _ := e.Info()
		var size int64
		var mtime time.Time
		if fi != nil {
			size = fi.Size()
			mtime = fi.ModTime()
		}
		display := stripExeSuffix(name)
		tools = append(tools, ToolInfo{
			Name:    display,
			Path:    filepath.Join(dir, name),
			Source:  source,
			Size:    size,
			ModTime: mtime,
		})
	}
	return tools
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

// PrintToolsTable prints tools in table format to the provided writer.
// This is the pkg/tools exported replacement for the previous cmd.printToolsTable.
func PrintToolsTable(w io.Writer, tools []ToolInfo, verbose bool) error {
	if verbose {
		headers := []string{"name", "source", "size", "modified", "path"}
		rows := make([][]string, 0, len(tools))
		for _, t := range tools {
			rows = append(rows, []string{
				t.Name,
				string(t.Source),
				formatSize(t.Size),
				t.ModTime.Format("2006-01-02 15:04"),
				t.Path,
			})
		}
		if err := style.PrintTable(w, headers, rows, 0); err != nil {
			return fmt.Errorf("failed to print tools list in table format: %w", err)
		}
		return nil
	}

	headers := []string{"name", "source", "path"}
	rows := make([][]string, 0, len(tools))
	for _, t := range tools {
		rows = append(rows, []string{t.Name, string(t.Source), t.Path})
	}
	if err := style.PrintTable(w, headers, rows, 0); err != nil {
		return fmt.Errorf("failed to print tools list in table format: %w", err)
	}
	return nil
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1fK", float64(size)/1024)
	}
	return fmt.Sprintf("%.1fM", float64(size)/(1024*1024))
}
