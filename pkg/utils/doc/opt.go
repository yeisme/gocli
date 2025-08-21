package doc

import (
	"fmt"
)

// Style 定义文档渲染的样式
type Style string

const (
	// StyleMarkdown 表示 Markdown 风格
	StyleMarkdown Style = "markdown"
	// StyleHTML 表示 HTML 风格
	StyleHTML Style = "html"
	// StylePlain 表示 Plain 风格
	StylePlain Style = "plain"
	// StyleJSON 表示 JSON 风格
	StyleJSON Style = "json"
	// StyleYAML 表示 YAML 风格
	StyleYAML Style = "yaml"
)

// Mode 定义文档生成的模式
type Mode string

const (
	// ModeGodoc Godoc 风格
	ModeGodoc Mode = "godoc"
	// ModeMarkdown Markdown 风格
	ModeMarkdown Mode = "markdown"
)

// Options 用于配置文档命令选项
type Options struct {

	// Output 指定生成文档的输出路径（文件或目录），为空则输出到 stdout 或默认位置
	Output string `mapstructure:"output"`

	// Style 渲染风格，使用什么样式渲染（markdown, html, plain）
	Style Style `mapstructure:"style"`

	// Mode 指定文档渲染模式 (godoc, markdown, etc.)
	Mode Mode `mapstructure:"mode"`

	// IncludePrivate 是否包含非导出（private）符号
	IncludePrivate bool `mapstructure:"include_private"`

	// IncludeTests 是否包含 *_test.go 文件（默认 false，不包含）
	IncludeTests bool `mapstructure:"include_tests"`

	// TOC 是否生成目录 (table of contents)
	TOC bool `mapstructure:"toc"`

	// Verbose 是否开启详细日志输出
	Verbose bool `mapstructure:"verbose"`

	// Exclude 列表，排除不需要生成文档的包或路径
	Exclude []string `mapstructure:"exclude"`

	// Theme 用于指定渲染时的主题 (例如 "dracula", "dark", "light")
	Theme string `mapstructure:"theme"`

	// Width 用于指定渲染的宽度，0 表示自动检测终端宽度
	Width int `mapstructure:"width"`

	// Detailed 详细模式，是否输出更详细的文档信息，仅在 godoc 模式下有效，用于更详细的文档输出
	Detailed bool `mapstructure:"detailed"`
}

// Validate 检查 Options 的基本有效性
func (o Options) Validate() error {
	// Debug: validate options
	log.Debug().Str("style", string(o.Style)).Str("mode", string(o.Mode)).Bool("verbose", o.Verbose).Msg("Options.Validate called")
	if !o.Style.IsValid() {
		return fmt.Errorf("doc: invalid style: %s", o.Style)
	}
	if !o.Mode.IsValid() {
		return fmt.Errorf("doc: invalid mode: %s", o.Mode)
	}
	return nil
}

// IsValid 返回 Style 是否是已知值
func (s Style) IsValid() bool {
	switch s {
	case StyleMarkdown, StyleHTML, StylePlain, StyleJSON, StyleYAML:
		return true
	}
	return false
}

// IsValid 返回 Mode 是否是已知值
func (m Mode) IsValid() bool {
	switch m {
	case ModeGodoc, ModeMarkdown:
		return true
	}
	return false
}
