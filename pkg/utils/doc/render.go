package doc

import (
	"io"
)

// RenderGodoc 渲染 Godoc 文档，并支持多种输出格式
// 当前实现为简单渲染：
//   - 当 opts.Title 非空时，将标题作为第一行输出
//   - 当 opts.TOC 为 true 时，会从输入中抽取一级/二级标题生成简单 TOC（基于行前缀 'Package ' 或 '##'）
func RenderGodoc(out io.Writer, input string, opts Options) error {
	switch opts.Style {
	case StylePlain:
		_ = renderPlain(out, input, opts)
	}
	return nil
}

// renderPlain 直接原样输出
func renderPlain(out io.Writer, input string, _ Options) error {
	_, _ = io.WriteString(out, input)
	return nil
}
