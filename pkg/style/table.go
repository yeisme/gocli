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
	// 终端宽度
	termWidth := detectTerminalWidth(w)
	if termWidth <= 0 {
		termWidth = 80
	}

	// 计算基于内容的“自然宽度”（尽量贴合字符串长度）
	naturalWidth := calcNaturalTableWidth(headers, rows)

	// 选择最终宽度：
	// - 若调用方指定了 width>0，尊重该宽度
	// - 否则在终端宽度与自然宽度之间取较小者
	if width <= 0 {
		if naturalWidth < termWidth {
			width = naturalWidth
		} else {
			width = termWidth
		}
	}

	re := lipgloss.NewRenderer(w)
	baseStyle := re.NewStyle().Padding(0, 1)
	headerStyle := baseStyle.Foreground(lipgloss.Color("252")).Bold(true)

	capitalizeHeaders := func(headers []string) []string {
		for i := range headers {
			headers[i] = strings.ToUpper(headers[i])
		}
		return headers
	}

	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("238"))).
		Headers(capitalizeHeaders(headers)...).
		Width(width).
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return baseStyle
		})

	_, err := fmt.Fprintln(w, tbl)
	return err
}

// detectTerminalWidth 尝试从 writer 获取终端宽度，失败则返回 0
func detectTerminalWidth(w io.Writer) int {
	// 优先使用文件描述符
	if f, ok := w.(*os.File); ok {
		if fw := f.Fd(); fw > 0 {
			if cols, _, err := xterm.GetSize(fw); err == nil && cols > 0 {
				return cols
			}
		}
	}
	// 尝试从环境变量读取（例如某些环境会设置 COLUMNS）
	if v := os.Getenv("COLUMNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

// 计算表格的“自然宽度”：
// = 各列最大内容宽度之和 + 每列左右 padding(2) + 边框竖线数量(列数+1)
// 使用 runewidth 以兼容中英文/emoji 宽度。
func calcNaturalTableWidth(headers []string, rows [][]string) int {
	if len(headers) == 0 {
		return 0
	}
	cols := len(headers)
	maxW := make([]int, cols)
	// 统计表头
	for i := 0; i < cols; i++ {
		if i < len(headers) {
			if w := runewidth.StringWidth(headers[i]); w > maxW[i] {
				maxW[i] = w
			}
		}
	}
	// 统计行内容
	for _, r := range rows {
		for i := 0; i < cols && i < len(r); i++ {
			// 多行单元格按最长行计算
			parts := strings.Split(r[i], "\n")
			for _, p := range parts {
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
	// 每列样式 padding: Padding(0,1) -> 左右各 1，共 2
	paddingSum := 2 * cols
	// 边框竖线：列数+1（例如 | a | b |）
	borders := cols + 1
	return contentSum + paddingSum + borders
}
