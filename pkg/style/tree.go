package style

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

// TreeNode 定义了用于构建树的数据结构
type TreeNode struct {
	Text     string
	Children []TreeNode
}

// PrintTree 用于渲染一个带有主题样式的树形结构到指定的 writer
// 它接收一个 TreeNode 结构作为数据源
func PrintTree(w io.Writer, rootNode TreeNode) error {
	// 定义各部分样式
	rootStyle := lipgloss.NewStyle().Foreground(ColorAccentText).Bold(true)
	itemStyle := lipgloss.NewStyle().Foreground(ColorText)
	enumeratorStyle := lipgloss.NewStyle().Foreground(ColorBorder)

	// 递归函数，用于将我们的 TreeNode 结构转换为 lipgloss/tree 对象
	var buildLipglossTree func(TreeNode) *tree.Tree
	buildLipglossTree = func(node TreeNode) *tree.Tree {
		// 为当前节点创建一个 lipgloss 树
		t := tree.New().Root(node.Text)

		// 遍历所有子节点，递归构建并添加到当前树中
		var children []any
		for _, child := range node.Children {
			children = append(children, buildLipglossTree(child))
		}
		t.Child(children...)

		return t
	}

	// 构建最终的 lipgloss 树并应用样式
	t := buildLipglossTree(rootNode).
		Enumerator(tree.RoundedEnumerator). // 使用圆角连接符
		RootStyle(rootStyle).
		ItemStyle(itemStyle).
		EnumeratorStyle(enumeratorStyle)

	_, err := fmt.Fprintln(w, t)
	return err
}
