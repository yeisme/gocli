// Package count 提供代码统计的工具函数
package count

import (
	"context"

	"github.com/yeisme/gocli/pkg/models"
)

// Options 用于控制统计行为与范围
// 所有字段均为可选，零值表示采用实现的默认策略
type Options struct {
	// 过滤与遍历
	Include          []string // 仅统计匹配这些 glob 的路径（优先级高于 Exclude）
	Exclude          []string // 排除匹配这些 glob 的路径（如 vendor、.git、node_modules 等）
	RespectGitignore bool     // 是否遵循 .gitignore
	FollowSymlinks   bool     // 是否跟随符号链接
	MaxFileSizeBytes int64    // 超过该大小的文件将被跳过（0 表示不限制）

	// 并发控制
	Concurrency int // 并发文件处理数量（<=0 表示由实现决定）

	// 统计项开关（关闭可加速）
	WithFunctions bool // 统计函数数量（若实现支持）
	WithStructs   bool // 统计结构体数量（若实现支持）

	// 结果细节
	WithFileDetails     bool // 填充 AnalysisResult.Files 列表
	WithLanguageDetails bool // 填充 LanguageStats.Files 列表

	// 语言特定信息（例如 Go 的包名/导入）
	WithLanguageSpecific bool
}

// File 单文件统计接口
type File interface {
	// CountSingleFile 统计单个文件，返回该文件的详细信息
	CountSingleFile(ctx context.Context, filePath string, opts Options) (*models.FileInfo, error)

	// 统计代码行数
	CountLines(ctx context.Context, filePath string) (int, error)
	CountBlanks(ctx context.Context, filePath string) (int, error)
	CountComments(ctx context.Context, filePath string) (int, error)
}

// GoFile Go 语言专属统计接口
type GoFile interface {
	// Go 语言专属：函数与结构体数量
	// CountGoDetails 统计 Go 文件特有的信息
	CountGoDetails(ctx context.Context, filePath string) (*models.GoDetails, error)

	CountFunctions(ctx context.Context, filePath string) (int, error)
	CountStructs(ctx context.Context, filePath string) (int, error)
}

// Project 项目统计接口
type Project interface {
	// CountAllFiles 遍历项目，返回每个文件的统计结果切片
	// 若 opts.WithFileDetails 或 opts.WithLanguageDetails 被关闭，实现仍可返回基本信息
	CountAllFiles(ctx context.Context, root string, opts Options) ([]models.FileInfo, error)

	// CountProjectSummary 返回项目整体分析结果
	// 建议实现中复用 CountAllFiles 以构建 models.AnalysisResult
	CountProjectSummary(ctx context.Context, root string, opts Options) (*models.AnalysisResult, error)
}

var (
	// ExtToLang 创建扩展名到语言的映射表
	ExtToLang = map[string]string{
		".go":  "Go",
		".js":  "JavaScript",
		".ts":  "TypeScript",
		".jsx": "JSX",
		".tsx": "TSX",
		// Python
		".py":  "Python",
		".pyi": "Python",

		".java": "Java",

		// C/C++ 系列
		".c":   "C",
		".cxx": "C++",
		".cc":  "C++",
		".cpp": "C++",
		".h":   "C Header",
		".hpp": "C++ Header",

		".rs":    "Rust",
		".rb":    "Ruby",
		".cs":    "C#",
		".swift": "Swift",
		".kt":    "Kotlin",
		".scala": "Scala",
		// Shell
		".sh":     "Shell",
		".bash":   "Shell",
		".zsh":    "Shell",
		".fish":   "Shell",
		".ps1":    "PowerShell",
		".sql":    "SQL",
		".html":   "HTML",
		".htm":    "HTML",
		".xml":    "XML",
		".css":    "CSS",
		".scss":   "SCSS",
		".sass":   "SASS",
		".less":   "LESS",
		".vue":    "Vue",
		".svelte": "Svelte",
		".yml":    "YAML",
		".yaml":   "YAML",
		".json":   "JSON",
		".toml":   "TOML",
		".ini":    "INI",
		".cfg":    "INI",
		".conf":   "INI",
		".md":     "Markdown",
		".txt":    "Text",
	}

	// LangToComment 语言到注释风格的映射
	LangToComment = map[string]commentStyle{
		"Go":         {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"Java":       {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"JavaScript": {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"TypeScript": {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"Python":     {single: []string{"#"}},
		"Ruby":       {single: []string{"#"}},
		"Rust":       {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"Swift":      {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"Shell":      {single: []string{"#"}},
		"HTML":       {single: nil, blockStart: "<!--", blockEnd: "-->"},
		"SQL":        {single: []string{"--"}, blockStart: "/*", blockEnd: "*/"},
		"Markdown":   {single: []string{"<!--"}, blockStart: "<!--", blockEnd: "-->"},
		"INI":        {single: []string{"#"}},
		"YAML":       {single: []string{"#"}},
		"JSON":       {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"XML":        {single: nil, blockStart: "<!--", blockEnd: "-->"},
		"CSS":        {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"SCSS":       {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"SASS":       {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"LESS":       {single: []string{"//"}, blockStart: "/*", blockEnd: "*/"},
		"Vue":        {single: []string{"<!--"}, blockStart: "<!--", blockEnd: "-->"},
		"Svelte":     {single: []string{"<!--"}, blockStart: "<!--", blockEnd: "-->"},
		"PowerShell": {single: []string{"#"}, blockStart: "<#", blockEnd: "#>"},
	}
)
