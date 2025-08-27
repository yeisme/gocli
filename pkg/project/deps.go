package project

import (
	"fmt"
	"io"
	"sort"

	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/deps"
)

// DepsOptions 定义了 `gocli project deps` 的各类选项，用于覆盖大部分 `go mod` 能力：
//   - 输出控制: Graph/Tree/JSON
//   - 版本更新: Update (透传 -u)
//   - 子命令包装: Tidy/Vendor/Download/Verify/Why 及其附加开关
type DepsOptions struct {
	// 输出样式
	Graph bool // 生成依赖关系图
	Tree  bool // 生成依赖树
	JSON  bool // JSON 输出格式

	Update  bool // 检查可用的更新
	Verbose bool

	// extra go mod subcommands
	Tidy      bool // go mod tidy
	Vendor    bool // go mod vendor
	Download  bool // go mod download
	Verify    bool // go mod verify
	Why       bool // go mod why
	WhyModule bool // go mod why -m
	WhyVendor bool // go mod why -vendor
}

// RunDeps 根据传入的 DepsOptions 执行依赖相关操作，并将结果写入 out
//
// 行为优先级:
//  1. 若开启 Tidy/Vendor/Download/Verify/Why，其对应的 `go mod` 子命令将被优先执行并返回；
//  2. 其次若开启 Tree/Graph：
//     - Tree: 基于 `go mod graph` 构建 DAG，并以树形样式渲染；
//     - Graph: 直接输出 `go mod graph` 的原始文本；
//  3. 其他情况下，默认执行 `go list -m`（可加 -json、-u），args 作为目标（默认 all）
//
// 参数:
//   - options: 控制输出风格与子命令行为；
//   - out: 结果输出 writer；
//   - args: 目标模块/模式（如 all、std、具体 import 路径）
//
// 返回:
//   - error: 命令执行或解析过程中的错误
func RunDeps(options DepsOptions, out io.Writer, args []string) error {
	// 1) 优先处理显式 go mod 子命令
	if handled, err := handleGoModSubcommands(options, out, args); handled || err != nil {
		return err
	}

	// 2) 依赖树视图
	if options.Tree {
		return renderDepsTree(out)
	}

	// 3) 原始依赖图
	if options.Graph {
		return printRawGraph(out)
	}

	// 4) 默认：go list -m
	return runGoList(options, out, args)
}

// handleGoModSubcommands 处理 go mod 类子命令；若已处理，返回 handled=true
func handleGoModSubcommands(options DepsOptions, out io.Writer, args []string) (bool, error) {
	switch {
	case options.Tidy:
		output, err := deps.RunGoModTidy()
		if err != nil {
			return true, err
		}
		fmt.Fprint(out, output)
		return true, nil
	case options.Vendor:
		output, err := deps.RunGoModVendor()
		if err != nil {
			return true, err
		}
		fmt.Fprint(out, output)
		return true, nil
	case options.Download:
		output, err := deps.RunGoModDownload()
		if err != nil {
			return true, err
		}
		fmt.Fprint(out, output)
		return true, nil
	case options.Verify:
		output, err := deps.RunGoModVerify()
		if err != nil {
			return true, err
		}
		fmt.Fprint(out, output)
		return true, nil
	case options.Why:
		output, err := deps.RunGoModWhy(args, struct{ Module, Vendor bool }{Module: options.WhyModule, Vendor: options.WhyVendor})
		if err != nil {
			return true, err
		}
		fmt.Fprint(out, output)
		return true, nil
	default:
		return false, nil
	}
}

// renderDepsTree 通过 `go mod graph` 构建 DAG，并以树形格式渲染到 out
func renderDepsTree(out io.Writer) error {
	raw, err := deps.RunGoModGraph()
	if err != nil {
		return err
	}
	g, err := deps.ParseGoModGraph(raw)
	if err != nil {
		return err
	}

	roots := findRoots(g)
	// 构建树节点
	var rootNode style.TreeNode
	if len(roots) == 1 {
		rootNode = buildTreeNode(g, roots[0])
	} else {
		// 多个根时使用合成根
		sort.Slice(roots, func(i, j int) bool { return roots[i].Path < roots[j].Path })
		children := make([]style.TreeNode, 0, len(roots))
		for _, r := range roots {
			children = append(children, buildTreeNode(g, r))
		}
		rootNode = style.TreeNode{Text: "modules", Children: children}
	}
	return style.PrintTree(out, rootNode)
}

// findRoots 尝试推断 DAG 的根模块集合
func findRoots(g *deps.Graph) []deps.Module {
	var roots []deps.Module
	// 1) 没有父节点的模块
	for _, m := range g.Modules() {
		if len(g.Parents(m.ID())) == 0 {
			roots = append(roots, m)
		}
	}
	// 2) 回退：无版本且有子节点
	if len(roots) == 0 {
		for _, m := range g.Modules() {
			if m.Version == "" && len(g.Children(m.ID())) > 0 {
				roots = append(roots, m)
			}
		}
	}
	// 3) 最后回退：按路径排序，取第一个
	if len(roots) == 0 {
		mods := g.Modules()
		sort.Slice(mods, func(i, j int) bool { return mods[i].Path < mods[j].Path })
		if len(mods) > 0 {
			roots = []deps.Module{mods[0]}
		}
	}
	return roots
}

// buildTreeNode 递归构建 TreeNode，保证子节点稳定排序
func buildTreeNode(g *deps.Graph, m deps.Module) style.TreeNode {
	label := m.Path
	if m.Version != "" {
		label = label + "@" + m.Version
	}
	children := g.Children(m.ID())
	sort.Slice(children, func(i, j int) bool {
		if children[i].Path == children[j].Path {
			return children[i].Version < children[j].Version
		}
		return children[i].Path < children[j].Path
	})
	nodes := make([]style.TreeNode, 0, len(children))
	for _, c := range children {
		nodes = append(nodes, buildTreeNode(g, c))
	}
	return style.TreeNode{Text: label, Children: nodes}
}

// printRawGraph 输出 `go mod graph` 的原始文本
func printRawGraph(out io.Writer) error {
	raw, err := deps.RunGoModGraph()
	if err != nil {
		return err
	}
	fmt.Fprint(out, raw)
	return nil
}

// runGoList 执行 `go list -m`，可选 json 与 update
func runGoList(options DepsOptions, out io.Writer, args []string) error {
	output, err := deps.RunGoModList(args, struct {
		JSON   bool
		Update bool
	}{
		JSON:   options.JSON,
		Update: options.Update,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s", output)
	return nil
}
