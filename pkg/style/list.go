package style

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
)

// PrintList 用于渲染一个带有主题样式的列表到指定的 writer
// items 参数支持嵌套，可以传入另一个 list.New() 对象来创建子列表
func PrintList(w io.Writer, items ...any) error {
	// 为编号符号定义样式: 使用主题强调色
	enumeratorStyle := lipgloss.NewStyle().
		Foreground(ColorAccentPrimary).
		MarginRight(1)

	// 为列表项文本定义样式: 使用主要文本颜色
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

// PrintPackageList 针对 Go 包列表进行渲染:
//   - 普通包使用主文本色
//   - 以 .test 结尾的测试包用方括号包裹并使用 ColorBorder 色
//   - 已经使用 [] 包裹的也使用测试包样式（例如用户已有特殊标记）
//   - 使用与 PrintList 一致的枚举 bullet 与强调色
func PrintPackageList(w io.Writer, pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	bulletStyle := lipgloss.NewStyle().Foreground(ColorAccentPrimary).MarginRight(1)
	normalStyle := lipgloss.NewStyle().Foreground(ColorText)
	testStyle := lipgloss.NewStyle().Foreground(ColorBorder)

	var b strings.Builder
	bullet := bulletStyle.Render(" •")
	for _, p := range pkgs {
		if p == "" {
			continue
		}

		// 情况 1: 纯测试包（以 .test 结尾且无额外前缀部分） => 整行使用测试样式
		if strings.HasSuffix(p, ".test") && !strings.Contains(p, " [") {
			b.WriteString(bullet)
			b.WriteString(testStyle.Render(p))
			b.WriteByte('\n')
			continue
		}

		// 情况 2: 含有中括号附加测试包，如:
		//   github.com/xxx/pkg [github.com/xxx/pkg.test]
		// 需求: 仅中括号部分用测试样式，前缀保持普通样式；可能存在多个中括号片段时逐段处理。
		if strings.Contains(p, "[") && strings.Contains(p, "]") {
			var lineBuilder strings.Builder
			remain := p
			for {
				start := strings.Index(remain, "[")
				if start < 0 { // 无更多括号
					if remain != "" {
						lineBuilder.WriteString(normalStyle.Render(remain))
					}
					break
				}
				// 前缀（普通样式）
				prefix := remain[:start]
				if prefix != "" {
					lineBuilder.WriteString(normalStyle.Render(prefix))
				}
				remain = remain[start:]
				end := strings.Index(remain, "]")
				if end < 0 { // 没有闭合，整体按普通样式输出剩余
					lineBuilder.WriteString(normalStyle.Render(remain))
					remain = ""
					break
				}
				segment := remain[:end+1] // 包含 ]
				lineBuilder.WriteString(testStyle.Render(segment))
				remain = remain[end+1:]
			}
			b.WriteString(bullet)
			b.WriteString(lineBuilder.String())
			b.WriteByte('\n')
			continue
		}

		// 情况 3: 普通包
		b.WriteString(bullet)
		b.WriteString(normalStyle.Render(p))
		b.WriteByte('\n')
	}
	_, err := fmt.Fprint(w, b.String())
	return err
}
