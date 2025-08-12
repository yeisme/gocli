package style

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Formatter 描述一个 formatter 及其说明
type Formatter struct {
	Name        string
	Description string
	Enabled     bool
}

// PrintHeading 打印一个区块标题
func PrintHeading(w io.Writer, title string) error {
	style := lipgloss.NewStyle().
		Foreground(ColorAccentText).
		Background(ColorAccentPrimary).
		Bold(true).
		Padding(0, 1)
	_, err := fmt.Fprintln(w, style.Render(strings.ToUpper(title)))
	return err
}

// PrintFormatterList 以对齐的方式打印 formatter 列表
func PrintFormatterList(w io.Writer, formatters []Formatter) error {
	if len(formatters) == 0 {
		return nil
	}
	// 计算最大名称长度用于对齐
	maxName := 0
	for _, f := range formatters {
		if l := len(f.Name); l > maxName {
			maxName = l
		}
	}

	nameEnabled := lipgloss.NewStyle().Foreground(ColorAccentPrimary).Bold(true)
	nameDisabled := lipgloss.NewStyle().Foreground(ColorDanger)
	descStyle := lipgloss.NewStyle().Foreground(ColorText)

	for _, f := range formatters {
		name := f.Name
		if f.Enabled {
			name = nameEnabled.Render(name)
		} else {
			name = nameDisabled.Render(name)
		}
		padding := strings.Repeat(" ", maxName-len(f.Name))
		line := fmt.Sprintf("  %s%s  %s", name, padding, descStyle.Render(f.Description))
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
