package tools

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/spf13/viper"
)

type (
	// InstallToolsInfo 定义了内置工具的信息
	InstallToolsInfo struct {
		Name string `mapstructure:"name" jsonschema:"title=Name,description=Tool identifier"`
		// URL: go install 的模块路径（可带 @version），与 CloneURL 互斥；保留兼容
		URL string `mapstructure:"url" jsonschema:"description=Go install module path (may include @version); mutually exclusive with CloneURL,nullable"`
		// CloneURL: 通过 git clone 构建时使用的仓库地址，可带 #ref（tag/branch/commit）
		CloneURL string `mapstructure:"clone_url" jsonschema:"description=Repository URL used for git clone; may include #ref (tag/branch/commit),nullable"`

		// Build: clone 模式下的构建方式，例如 "make"、"goreleaser"（默认 make）
		Build string `mapstructure:"build" jsonschema:"description=Build method for clone mode, e.g. make or goreleaser,nullable"`
		// MakeTarget: make 模式下要执行的目标名（可为空执行默认目标）
		MakeTarget string `mapstructure:"make_target" jsonschema:"description=Make target to run (empty for default),nullable"`
		// WorkDir: 在仓库中的子目录进行构建（为空则仓库根目录）
		WorkDir string `mapstructure:"work_dir" jsonschema:"description=Subdirectory within the repository to build in (empty means repo root),nullable"`
		// BinDirs: 构建产物所在目录（相对 WorkDir/仓库根）；为空则按构建器默认（goreleaser -> dist）
		BinDirs []string `mapstructure:"bin_dirs" jsonschema:"description=Directories containing build artifacts (relative to WorkDir/repo root),nullable,uniqueItems"`
		// Env: 该工具需要的额外环境变量（如 CGO_ENABLED=1）
		Env []string `mapstructure:"env" jsonschema:"description=Additional environment variables required by the tool (e.g., CGO_ENABLED=1),nullable,uniqueItems"`
		// GoreleaserConfig: goreleaser 构建时的配置文件路径（相对 WorkDir/仓库根）
		GoreleaserConfig string `mapstructure:"goreleaser_config" jsonschema:"description=Path to goreleaser config file (relative to WorkDir/repo root),nullable"`

		InstallType *InstallType `mapstructure:"install_type" jsonschema:"description=Platform constraints for installation; contains name/os/arch; may be null,nullable"`
		// BinaryName 指定该工具生成的二进制文件名（可选）；
		BinaryName string `mapstructure:"binary_name" jsonschema:"description=Optional binary filename produced by the tool; if empty default rules apply (e.g., go install),nullable"`
	}

	// InstallType 定义了内置工具的安装类型
	InstallType struct {
		Name string `mapstructure:"name" jsonschema:"description=Install type name, e.g. go_install or clone,nullable"`
		OS   string `mapstructure:"os" jsonschema:"description=Restricted OS name, e.g. linux/darwin/windows,nullable"`
		Arch string `mapstructure:"arch" jsonschema:"description=Restricted CPU architecture, e.g. amd64/arm64,nullable"`
	}
)

//go:embed embed_tools.json
var embedTools []byte

// BuiltinTools 内部维护工具的名称和安装路径（既可 go install，也可 clone 构建）
var BuiltinTools map[string]InstallToolsInfo

var builtinInitOnce sync.Once

func init() {
	// 使用 sync.Once 确保只初始化一次
	builtinInitOnce.Do(func() {
		// 尝试从嵌入的 JSON 初始化
		if len(embedTools) > 0 {
			v := viper.New()
			v.SetConfigType("json")
			if err := v.ReadConfig(bytes.NewReader(embedTools)); err == nil {
				var m map[string]InstallToolsInfo
				if err := v.Unmarshal(&m); err == nil && len(m) > 0 {
					BuiltinTools = m
					return
				}
			}
		}
		// 如果嵌入资源不可用或为空，则初始化为空映射，后续可通过 LoadUserTools 合并用户定义
		BuiltinTools = make(map[string]InstallToolsInfo)
	})
}

// LoadUserTools 从指定的 JSON 文件加载用户自定义的工具表并合并到全局 BuiltinTools 中
// 如果键存在则会覆盖已有条目
func LoadUserTools(path string) error {
	if path == "" {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, ".gocli", "tools.json")
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// 尝试基于扩展名或内容识别格式
	tryTypes := []string{}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json", ".jsonc":
		tryTypes = append(tryTypes, "json")
	case ".yaml", ".yml":
		tryTypes = append(tryTypes, "yaml")
	case ".toml":
		tryTypes = append(tryTypes, "toml")
	default:
		// 内容探测（非常简化）
		trim := bytes.TrimSpace(data)
		if len(trim) > 0 && (trim[0] == '{' || trim[0] == '[') {
			tryTypes = append(tryTypes, "json")
		}
		// yaml 最后再试（yaml 解析器很宽松）
		tryTypes = append(tryTypes, "yaml", "toml")
	}
	// 去重
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(tryTypes))
	for _, t := range tryTypes {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		uniq = append(uniq, t)
	}

	var lastErr error
	var m map[string]InstallToolsInfo
	for _, typ := range uniq {
		v := viper.New()
		v.SetConfigType(typ)
		if err := v.ReadConfig(bytes.NewReader(data)); err != nil {
			lastErr = err
			continue
		}
		if err := v.Unmarshal(&m); err != nil {
			lastErr = err
			continue
		}
		// success
		lastErr = nil
		break
	}
	if lastErr != nil {
		return fmt.Errorf("parse user tools '%s' failed: %w", path, lastErr)
	}
	if len(m) == 0 {
		return errors.New("user tools file parsed but empty")
	}
	if BuiltinTools == nil {
		BuiltinTools = make(map[string]InstallToolsInfo)
	}
	// 检查与现有 Name 冲突（不同 key 相同 Name）
	nameIndex := map[string]string{}
	for k, v := range BuiltinTools {
		nameIndex[v.Name] = k
	}
	for k, v := range m {
		if prevKey, ok := nameIndex[v.Name]; ok && prevKey != k {
			// 仅输出到 stderr（不返回错误）
			fmt.Fprintf(os.Stderr, "[gocli][tools] warning: duplicate tool name '%s' (existing key '%s', new key '%s')\n", v.Name, prevKey, k)
		}
	}
	maps.Copy(BuiltinTools, m)
	return nil
}

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
		// 返回指向该值副本的指针以避免引用循环变量或后续修改
		t := tool
		return &t
	}
	return nil
}

// FindToolsFuzzy 在内置工具表（以及用户合并的表）中进行模糊搜索
// 返回按 Name 排序的结果切片（可能为空）
func FindToolsFuzzy(query string, paths []string) []InstallToolsInfo {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}

	// load user tools from paths
	for _, p := range paths {
		_ = LoadUserTools(p)
	}
	if BuiltinTools == nil {
		BuiltinTools = make(map[string]InstallToolsInfo)
	}

	// prepare candidates
	type pair struct {
		key string
		t   InstallToolsInfo
		s   string
	}
	var list []pair
	for k, t := range BuiltinTools {
		// combine important fields into a single searchable string
		s := strings.ToLower(strings.Join([]string{
			t.Name,
			t.URL,
			t.CloneURL,
			t.Build,
			t.MakeTarget,
			t.WorkDir,
			strings.Join(t.BinDirs, " "),
			strings.Join(t.Env, " "),
			t.BinaryName,
		}, " "))
		list = append(list, pair{key: k, t: t, s: s})
	}

	// use lithammer fuzzysearch to match against candidate strings
	var out []InstallToolsInfo
	lowerQ := strings.ToLower(q)
	for _, p := range list {
		// first try fuzzy.Match (better than simple substring for relevance)
		if fuzzy.Match(lowerQ, p.s) || strings.Contains(p.s, lowerQ) {
			out = append(out, p.t)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Search executes a search with optional interactive selection.
// If an exact or single fuzzy match is found, returns that item (non-nil) and nil slice.
// If multiple fuzzy matches exist and interactive==false, returns nil + matches.
// If interactive==true, launches interactive selector and returns selected item or error.
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
// 返回单个匹配 (非 nil) 或者多项匹配切片（长度>0）
func ResolveTool(name string, paths []string) (*InstallToolsInfo, []InstallToolsInfo) {
	// 精确查找（包含用户合并的表）
	if t := SearchTools(name, paths); t != nil {
		return t, nil
	}
	// 模糊搜索
	matches := FindToolsFuzzy(name, paths)
	if len(matches) == 1 {
		// 返回单项匹配的副本指针
		m := matches[0]
		return &m, nil
	}
	return nil, matches
}

// InteractiveSelect 使用 fuzzyfinder 在多个候选中交互选择一项
// 当用户取消或发生错误时返回非 nil 错误
func InteractiveSelect(matches []InstallToolsInfo) (*InstallToolsInfo, error) {
	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches to select")
	}
	idx, err := fuzzyfinder.Find(
		matches,
		func(i int) string {
			m := matches[i]
			return fmt.Sprintf("%s — %s %s", m.Name, m.URL, m.CloneURL)
		},
	)
	if err != nil {
		return nil, err
	}
	if idx < 0 || idx >= len(matches) {
		return nil, fmt.Errorf("invalid selection")
	}
	sel := matches[idx]
	return &sel, nil
}
