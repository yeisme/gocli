package style

import (
	"io"

	"github.com/charmbracelet/glamour"
)

// RenderMarkdown 渲染传入的 Markdown 文本并输出到指定 writer
// 与其它 style 包函数风格一致: 写入 w 并返回 error
// 基于终端宽度自动换行，最小宽度为 80，最大宽度为 120
//
// 参数:
//
//  1. w: 输出的 io.Writer
//  2. input: 要渲染的 Markdown 文本
//  3. width: 渲染的宽度
//  4. theme: 渲染时使用的主题 (例如 "dracula", "dark", "light" 等)
func RenderMarkdown(w io.Writer, input string, width int, theme string) error {
	if theme == "" {
		theme = "dracula"
	}
	// 解析并确定最终渲染宽度，优先级：显式参数(width>0) > 终端探测 > 默认80
	termWidth := detectTerminalWidth(w)
	if termWidth <= 0 {
		termWidth = 80
	}

	if width <= 0 {
		// 未传入宽度时，使用探测到的终端宽度
		width = termWidth
	}

	// 将宽度限制到 [80, 120]，且不超过探测到的终端宽度
	if width < 80 {
		width = 80
	}
	if width > 120 {
		if termWidth > 0 {
			width = min(120, termWidth)
		} else {
			width = 120
		}
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
		glamour.WithStandardStyle(theme),
		glamour.WithInlineTableLinks(true),
	)
	if err != nil {
		return err
	}

	out, err := r.Render(input)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, out)
	return err
}
