package deps

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/yeisme/gocli/pkg/utils/executor"
)

// Module 表示一个 Go 模块以及可选的版本号.
// 例如:
//
//	Path: github.com/yeisme/gocli, Version: ""
//	Path: github.com/charmbracelet/lipgloss, Version: v1.1.0
type Module struct {
	Path    string
	Version string
}

// ID 返回该模块的稳定标识，若包含版本则为 "path@version"；
// 与 `go mod graph` 的 token 格式一致.
func (m Module) ID() string {
	if m.Version == "" {
		return m.Path
	}
	return m.Path + "@" + m.Version
}

// DocURL 返回该模块在 pkg.go.dev 上的文档链接（包含版本时指向对应版本的页面）.
func (m Module) DocURL() string {
	if m.Version == "" {
		return fmt.Sprintf("https://pkg.go.dev/%s", m.Path)
	}
	return fmt.Sprintf("https://pkg.go.dev/%s@%s", m.Path, m.Version)
}

// Graph 表示 `go mod graph` 输出构建的依赖有向无环图（DAG）.
// 注意：多个父节点可能依赖同一个子节点，因此它不是树.
type Graph struct {
	// modules stores all unique modules by their ID (path[@version]).
	modules map[string]Module
	// edges maps parentID -> set(childID)
	edges map[string]map[string]struct{}
	// revEdges maps childID -> set(parentID)
	revEdges map[string]map[string]struct{}
}

// NewGraph 创建一个空图实例.
func NewGraph() *Graph {
	return &Graph{
		modules:  make(map[string]Module),
		edges:    make(map[string]map[string]struct{}),
		revEdges: make(map[string]map[string]struct{}),
	}
}

// AddEdge 向图中添加一条依赖边 parent -> child.
func (g *Graph) AddEdge(parent, child Module) {
	pID := parent.ID()
	cID := child.ID()
	if _, ok := g.modules[pID]; !ok {
		g.modules[pID] = parent
	}
	if _, ok := g.modules[cID]; !ok {
		g.modules[cID] = child
	}
	if _, ok := g.edges[pID]; !ok {
		g.edges[pID] = make(map[string]struct{})
	}
	g.edges[pID][cID] = struct{}{}

	if _, ok := g.revEdges[cID]; !ok {
		g.revEdges[cID] = make(map[string]struct{})
	}
	g.revEdges[cID][pID] = struct{}{}
}

// Children 返回给定模块 ID 的直接依赖列表.
func (g *Graph) Children(id string) []Module {
	set := g.edges[id]
	if len(set) == 0 {
		return nil
	}
	out := make([]Module, 0, len(set))
	for cid := range set {
		out = append(out, g.modules[cid])
	}
	return out
}

// Parents 返回给定模块 ID 的直接被依赖（父）列表.
func (g *Graph) Parents(id string) []Module {
	set := g.revEdges[id]
	if len(set) == 0 {
		return nil
	}
	out := make([]Module, 0, len(set))
	for pid := range set {
		out = append(out, g.modules[pid])
	}
	return out
}

// Modules 返回图中包含的所有模块.
func (g *Graph) Modules() []Module {
	out := make([]Module, 0, len(g.modules))
	for _, m := range g.modules {
		out = append(out, m)
	}
	return out
}

// Has 判断图中是否存在指定模块 ID.
func (g *Graph) Has(id string) bool { _, ok := g.modules[id]; return ok }

// ParseGoModGraph 解析 `go mod graph` 的多行文本输出并构建 Graph.
// 也可通过 ParseGoModGraphReader 从 io.Reader 解析.
func ParseGoModGraph(output string) (*Graph, error) {
	return ParseGoModGraphReader(strings.NewReader(output))
}

// ParseGoModGraphReader 按行解析 `go mod graph` 内容.
// 每行格式为："<parent> <child>".
func ParseGoModGraphReader(r io.Reader) (*Graph, error) {
	g := NewGraph()
	s := bufio.NewScanner(r)
	lineNo := 0
	for s.Scan() {
		lineNo++
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid go mod graph line %d: %q", lineNo, line)
		}
		parentTok := fields[0]
		childTok := fields[1]
		p := parseModuleToken(parentTok)
		c := parseModuleToken(childTok)
		g.AddEdge(p, c)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(g.modules) == 0 {
		return nil, errors.New("empty graph: no modules parsed")
	}
	return g, nil
}

// RunGoModGraph 执行 `go mod graph` 并返回其原始输出文本.
func RunGoModGraph(args ...string) (string, error) {
	base := []string{"mod", "graph"}
	if len(args) > 0 {
		base = append(base, args...)
	}
	return executor.NewExecutor("go", base...).Output()
}

// parseModuleToken 辅助函数：将 token（如 "github.com/foo/bar@v1.2.3" 或无版本）解析为 Module.
func parseModuleToken(tok string) Module {
	// Split only at the last '@' to be safe, though module paths normally don't contain '@'.
	if i := strings.LastIndex(tok, "@"); i >= 0 {
		return Module{Path: tok[:i], Version: tok[i+1:]}
	}
	return Module{Path: tok}
}
