package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/yeisme/gocli/pkg/style"
	"gopkg.in/yaml.v3"
)

// SearchTools 根据工具名称和路径查找工具信息
func SearchTools(name string, path []string) *InstallToolsInfo {
	// 尝试合并用户自定义工具表；如果用户文件不存在则忽略该错误并继续使用内置表
	for _, p := range path {
		_ = LoadUserTools(p)
	}

	// 保证 BuiltinTools 已初始化
	if BuiltinTools == nil {
		BuiltinTools = make(map[string]InstallToolsInfo)
	}

	if tool, ok := BuiltinTools[name]; ok {
		t := tool
		return &t
	}
	return nil
}

// FindToolsFuzzy 在内置工具表（以及用户合并的表）中进行模糊搜索
func FindToolsFuzzy(query string, paths []string) []InstallToolsInfo {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}
	for _, p := range paths {
		_ = LoadUserTools(p)
	}
	if BuiltinTools == nil {
		BuiltinTools = make(map[string]InstallToolsInfo)
	}
	type pair struct {
		key string
		t   InstallToolsInfo
		s   string
	}
	var list []pair
	for k, t := range BuiltinTools {
		s := strings.ToLower(strings.Join([]string{
			t.Name, t.URL, t.CloneURL, t.Build, t.MakeTarget, t.WorkDir, strings.Join(t.BinDirs, " "), strings.Join(t.Env, " "), t.BinaryName,
		}, " "))
		list = append(list, pair{key: k, t: t, s: s})
	}
	var out []InstallToolsInfo
	lowerQ := strings.ToLower(q)
	for _, p := range list {
		if fuzzy.Match(lowerQ, p.s) || strings.Contains(p.s, lowerQ) {
			out = append(out, p.t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Search executes a search with optional interactive selection.
func Search(name string, paths []string, interactive bool) (*InstallToolsInfo, []InstallToolsInfo, error) {
	bi, matches := ResolveTool(name, paths)
	if bi != nil {
		return bi, nil, nil
	}
	if len(matches) == 0 {
		return nil, nil, nil
	}
	if !interactive {
		return nil, matches, nil
	}
	sel, err := InteractiveSelect(matches)
	if err != nil {
		return nil, matches, err
	}
	return sel, nil, nil
}

// ResolveTool 查找工具：先精确查找配置表，若找不到则进行模糊搜索
func ResolveTool(name string, paths []string) (*InstallToolsInfo, []InstallToolsInfo) {
	if t := SearchTools(name, paths); t != nil {
		return t, nil
	}
	matches := FindToolsFuzzy(name, paths)
	if len(matches) == 1 {
		m := matches[0]
		return &m, nil
	}
	return nil, matches
}

// InteractiveSelect 使用 fuzzyfinder 在多个候选中交互选择一项
func InteractiveSelect(matches []InstallToolsInfo) (*InstallToolsInfo, error) {
	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches to select")
	}
	idx, err := fuzzyfinder.Find(matches, func(i int) string {
		m := matches[i]
		return fmt.Sprintf("%s — %s %s", m.Name, m.URL, m.CloneURL)
	})
	if err != nil {
		return nil, err
	}
	if idx < 0 || idx >= len(matches) {
		return nil, fmt.Errorf("invalid selection")
	}
	sel := matches[idx]
	return &sel, nil
}

// SearchCommandOptions 定义了搜索命令的选项
type SearchCommandOptions struct {
	Query     string
	Format    string
	JSON      bool
	YAML      bool
	Table     bool
	ConfigDir []string
}

// ExecuteSearchCommand 执行搜索命令
func ExecuteSearchCommand(opts SearchCommandOptions, outputWriter io.Writer) error {
	query := opts.Query
	fmtFlag := opts.Format
	setCount := 0
	if opts.Format != "" {
		setCount++
	}
	if opts.JSON {
		setCount++
	}
	if opts.YAML {
		setCount++
	}
	if opts.Table {
		setCount++
	}
	if setCount > 1 {
		return fmt.Errorf("only one of --format, --json, --yaml, --table may be specified")
	}
	if opts.JSON {
		fmtFlag = "json"
	} else if opts.YAML {
		fmtFlag = "yaml"
	} else if opts.Table {
		fmtFlag = "table"
	}
	if fmtFlag == "" {
		fmtFlag = "table"
	}
	if query == "" {
		if len(BuiltinTools) == 0 {
			return fmt.Errorf("no tools available")
		}
		all := make([]InstallToolsInfo, 0, len(BuiltinTools))
		for _, t := range BuiltinTools {
			all = append(all, t)
		}
		sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
		sel, err := InteractiveSelect(all)
		if err != nil {
			return fmt.Errorf("interactive select failed: %w", err)
		}
		return PrintSingleTool(sel, fmtFlag, outputWriter)
	}
	if bi := SearchTools(query, opts.ConfigDir); bi != nil {
		return PrintSingleTool(bi, fmtFlag, outputWriter)
	}
	matches := FindToolsFuzzy(query, opts.ConfigDir)
	if len(matches) == 0 {
		if fmtFlag == "json" || fmtFlag == "yaml" {
			fmt.Fprintln(outputWriter, "null")
		} else {
			return fmt.Errorf("tool not found: %s", query)
		}
		return nil
	}
	if len(matches) == 1 {
		m := matches[0]
		return PrintSingleTool(&m, fmtFlag, outputWriter)
	}
	return printMultipleTools(matches, fmtFlag, outputWriter)
}

func printMultipleTools(matches []InstallToolsInfo, fmtFlag string, out io.Writer) error {
	switch strings.ToLower(fmtFlag) {
	case "json":
		return json.NewEncoder(out).Encode(matches)
	case "yaml":
		return yaml.NewEncoder(out).Encode(matches)
	case "table":
		headers := []string{"Name", "URL", "CloneURL", "Build"}
		rows := make([][]string, 0, len(matches))
		for _, m := range matches {
			rows = append(rows, []string{m.Name, m.URL, m.CloneURL, m.Build})
		}
		return style.PrintTable(out, headers, rows, 0)
	default:
		return fmt.Errorf("unsupported format: %s", fmtFlag)
	}
}
