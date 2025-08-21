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
	termWidth := detectTerminalWidth(w)
	if termWidth <= 0 {
		termWidth = 80
	}
	// 如果传入 width <= 0 则使用探测到的终端宽度
	if width <= 0 {
		width = termWidth
	}
	// 将宽度限制到 [80, 120]
	if width < 80 {
		width = 80
	}
	if width > 120 {
		// 不要超过终端宽度
		width = min(120, termWidth)
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
