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
)

// PrintTable 用于标准化表格输出，支持自定义表头和内容
// width: 期望的表格宽度；当 width<=0 时自动探测终端宽度（失败则回退到80）。
func PrintTable(w io.Writer, headers []string, rows [][]string, width int) error {
	if width <= 0 {
		width = detectTerminalWidth(w)
		if width <= 0 {
			width = 80
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
