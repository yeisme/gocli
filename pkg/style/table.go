package style

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	xterm "github.com/charmbracelet/x/term"
	runewidth "github.com/mattn/go-runewidth"
)

// PrintTable 用于标准化表格输出，支持自定义表头和内容
// width: 期望的表格宽度；当 width<=0 时自动探测终端宽度（失败则回退到80）。
func PrintTable(w io.Writer, headers []string, rows [][]string, width int) error {
	termWidth := detectTerminalWidth(w)
	if termWidth <= 0 {
		termWidth = 80
	}
	naturalWidth := calcNaturalTableWidth(headers, rows)
	if width <= 0 {
		width = min(naturalWidth, termWidth)
	}

	re := lipgloss.NewRenderer(w)

	headerStyle := re.NewStyle().
		Foreground(ColorAccentText).
		Background(ColorAccentPrimary).
		Bold(true).
		Padding(0, 1).
		Align(lipgloss.Center)

	// 单元格基础样式: 左右内边距
	cellStyle := re.NewStyle().Padding(0, 1)

	// 奇数行样式: 浅灰色文字
	oddRowStyle := cellStyle.Foreground(ColorText)

	// 偶数行样式: 浅灰色文字，深灰色背景（形成斑马条纹）
	evenRowStyle := cellStyle.Foreground(ColorText).Background(ColorBackgroundAlternate)

	// 将表头文字转为大写
	capitalizeHeaders := func(headers []string) []string {
		for i := range headers {
			headers[i] = strings.ToUpper(headers[i])
		}
		return headers
	}

	tbl := table.New().
		// 设置总宽度
		Width(width).
		// 使用圆角边框
		Border(lipgloss.RoundedBorder()).
		// 设置边框颜色为深灰色
		BorderStyle(re.NewStyle().Foreground(ColorBorder)).
		// 设置表头
		Headers(capitalizeHeaders(headers)...).
		// 设置所有行数据
		Rows(rows...).
		// 使用 StyleFunc 为不同行应用不同样式
		StyleFunc(func(row, _ int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return headerStyle
			case row%2 == 0:
				// 偶数行
				return evenRowStyle
			default:
				// 奇数行
				return oddRowStyle
			}
		})

	_, err := fmt.Fprintln(w, tbl)
	return err
}

// detectTerminalWidth 尝试从 writer 获取终端宽度，失败则返回 0
func detectTerminalWidth(w io.Writer) int {
	if f, ok := w.(*os.File); ok {
		if fw := f.Fd(); fw > 0 {
			if cols, _, err := xterm.GetSize(fw); err == nil && cols > 0 {
				return cols
			}
		}
	}
	if v := os.Getenv("COLUMNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

// 计算表格的宽度
// = 各列最大内容宽度之和 + 每列左右 padding(2) + 边框竖线数量(列数+1)
// 使用 runewidth 以兼容中英文/emoji 宽度。
func calcNaturalTableWidth(headers []string, rows [][]string) int {
	if len(headers) == 0 {
		return 0
	}
	cols := len(headers)
	maxW := make([]int, cols)
	for i := 0; i < cols; i++ {
		if i < len(headers) {
			if w := runewidth.StringWidth(headers[i]); w > maxW[i] {
				maxW[i] = w
			}
		}
	}
	for _, r := range rows {
		for i := 0; i < cols && i < len(r); i++ {
			parts := strings.SplitSeq(r[i], "\n")
			for p := range parts {
				if w := runewidth.StringWidth(p); w > maxW[i] {
					maxW[i] = w
				}
			}
		}
	}
	contentSum := 0
	for _, w := range maxW {
		contentSum += w
	}
	paddingSum := 2 * cols
	borders := cols + 1
	return contentSum + paddingSum + borders
}
