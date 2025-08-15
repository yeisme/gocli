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

// PrintGoModUpdatesList 专用于渲染 `go list -m -u all` 的行列表。
// 规则：
//   - 仅包含 `vX.Y.Z`（没有方括号更新部分）的行，整体用绿色(ColorSuccess)；
//   - 包含 `current [next]` 形式的行：current 使用红色(ColorDanger)，方括号包含的 next 使用蓝色(ColorAccentPrimary)；
//   - 其余部分（模块名和空白）使用普通文本色(ColorText)。
//   - 每行前加与 PrintList 一致的 bullet。
func PrintGoModUpdatesList(w io.Writer, lines []string) error {
	if len(lines) == 0 {
		return nil
	}

	bulletStyle := lipgloss.NewStyle().Foreground(ColorAccentPrimary).MarginRight(1)
	normal := lipgloss.NewStyle().Foreground(ColorText)
	ok := lipgloss.NewStyle().Foreground(ColorSuccess)
	danger := lipgloss.NewStyle().Foreground(ColorDanger)
	next := lipgloss.NewStyle().Foreground(ColorAccentPrimary)

	var b strings.Builder
	bullet := bulletStyle.Render(" •")

	for _, raw := range lines {
		if strings.TrimSpace(raw) == "" {
			continue
		}

		line := raw
		// 是否包含 [next]
		lb := strings.Index(line, "[")
		rb := -1
		if lb >= 0 {
			rb = strings.Index(line[lb:], "]")
			if rb >= 0 {
				rb = lb + rb
			}
		}

		b.WriteString(bullet)

		if lb >= 0 && rb > lb {
			leftRaw := line[:lb]
			// 保留 leftRaw 的尾随空格用于还原与中括号之间的空白
			leftTrim := strings.TrimRight(leftRaw, " ")
			midSpaces := leftRaw[len(leftTrim):] // 版本和 [ 之间的原始空格（可能是一个或多个）

			// 在 leftTrim 中找到 current 版本（最后一个空格后的 token）
			ls := strings.LastIndex(leftTrim, " ")
			if ls < 0 || ls == len(leftTrim)-1 { // 解析失败，按普通文本处理
				b.WriteString(normal.Render(line))
				b.WriteByte('\n')
				continue
			}
			prefix := leftTrim[:ls+1]     // 含紧邻版本前的空格
			currentVer := leftTrim[ls+1:] // 旧版本
			nextWithBrackets := strings.TrimSpace(line[lb : rb+1])

			// 渲染：模块前缀(普通) + 旧版本(红) + 原空白 + [next](蓝)
			b.WriteString(normal.Render(prefix))
			b.WriteString(danger.Render(currentVer))
			if midSpaces == "" { // 至少保留一个空格
				b.WriteString(" ")
			} else {
				b.WriteString(midSpaces)
			}
			b.WriteString(next.Render(nextWithBrackets))
			b.WriteByte('\n')
			continue
		}

		// 无更新：取最后一个空格后为当前版本，之前为模块前缀
		lastSpace := strings.LastIndex(line, " ")
		if lastSpace <= 0 || lastSpace == len(line)-1 {
			b.WriteString(normal.Render(line))
			b.WriteByte('\n')
			continue
		}
		prefix := line[:lastSpace+1]
		currentVer := line[lastSpace+1:]
		b.WriteString(normal.Render(prefix))
		b.WriteString(ok.Render(currentVer))
		b.WriteByte('\n')
	}

	_, err := fmt.Fprint(w, b.String())
	return err
}
