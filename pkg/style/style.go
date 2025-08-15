// Package style 提供多种样式化输出功能
package style

import "github.com/charmbracelet/lipgloss"

// 定义一套颜色，方便管理和修改
const (
	// 主题强调色/品牌色，用于吸引注意力的元素，如表头背景
	ColorAccentPrimary = lipgloss.Color("#33A1FF")

	// 强调文本色，用于在强调背景(AccentPrimary)上显示的文本，以确保对比度
	ColorAccentText = lipgloss.Color("#FFFFFF")

	// 主要文本颜色，用于普通的数据行内容
	ColorText = lipgloss.Color("#E4E4E4")

	// 边框颜色，用于表格或容器的轮廓
	ColorBorder = lipgloss.Color("#444444")

	// 交替背景色，用于斑马条纹等，以区分相邻项目
	ColorBackgroundAlternate = lipgloss.Color("#303030")

	// 危险/禁用/错误强调色（例如 Disabled linters/formatters 名称）
	ColorDanger = lipgloss.Color("#FF5555")

	// 成功/通过/无更新 绿色
	ColorSuccess = lipgloss.Color("#22C55E")

	// JSON 高亮颜色
	ColorJSONKey    = lipgloss.Color("#55bcf4ff") // 键名
	ColorJSONValue  = ColorAccentText             // 字符串值
	ColorJSONNumber = lipgloss.Color("#d4ec19ff") // 数字
	ColorJSONBool   = lipgloss.Color("#dfab49ff") // 布尔
	ColorJSONNull   = lipgloss.Color("#6272A4")   // null
	ColorJSONPunct  = lipgloss.Color("#6B7280")   // 标点
)
