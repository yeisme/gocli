package style

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
)

// PrintList 用于渲染一个带有主题样式的列表到指定的 writer
// items 参数支持嵌套，可以传入另一个 list.New() 对象来创建子列表
func PrintList(w io.Writer, items ...any) error {
	// 为编号符号定义样式：使用主题强调色
	enumeratorStyle := lipgloss.NewStyle().
		Foreground(ColorAccentPrimary).
		MarginRight(1)

	// 为列表项文本定义样式：使用主要文本颜色
	itemStyle := lipgloss.NewStyle().
		Foreground(ColorText)

	// 创建列表并应用样式
	l := list.New(items...).
		Enumerator(list.Bullet). // 使用圆点作为编号
		EnumeratorStyle(enumeratorStyle).
		ItemStyle(itemStyle)

	_, err := fmt.Fprintln(w, l)
	return err
}
