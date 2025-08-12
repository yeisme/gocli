package models

// Stats 存储代码、注释和空行的计数，是一个可重用的基本单位
type Stats struct {
	Code     int `json:"code" yaml:"code"`         // 代码行数
	Comments int `json:"comments" yaml:"comments"` // 注释行数
	Blanks   int `json:"blanks" yaml:"blanks"`     // 空白行数
}

// FileInfo 存储单个文件的详细统计信息
type FileInfo struct {
	Path     string `json:"path" yaml:"path"`         // 文件相对于项目根目录的路径
	Language string `json:"language" yaml:"language"` // 文件所属的编程语言
	Stats    Stats  `json:"stats" yaml:"stats"`       // 该文件的行数统计

	// LanguageSpecific 用于存储特定语言的额外信息，提供扩展性
	// 对于 C/C++，可以存头文件依赖等
	LanguageSpecific any `json:"language_specific,omitempty" yaml:"language_specific,omitempty"`
}

// GoDetails 存储 Go 文件特有的信息
type GoDetails struct {
	PackageName string   `json:"package_name" yaml:"package_name"` // 包名
	Imports     []string `json:"imports" yaml:"imports"`           // 导入的依赖

	// 以下为可选统计项
	Functions int `json:"functions,omitempty" yaml:"functions,omitempty"` // 函数数量（按语言定义）
	Structs   int `json:"structs,omitempty" yaml:"structs,omitempty"`     // 结构体数量（按语言定义）

	LanguageStats
}

// LanguageStats 存储单一语言的聚合统计信息
type LanguageStats struct {
	FileCount int        `json:"file_count" yaml:"file_count"` // 该语言的文件总数
	Stats     Stats      `json:"stats" yaml:"stats"`           // 该语言的总行数统计
	Files     []FileInfo `json:"files,omitempty" yaml:"files,omitempty"`
}

// AnalysisResult 是最终分析结果的顶层结构体
type AnalysisResult struct {
	// Total 存储所有语言的总体统计
	Total LanguageStats `json:"total" yaml:"total"`

	// Languages 是一个映射，键是语言名称 (例如 "Go", "Markdown")，
	// 值是该语言的聚合统计信息使用指针可以方便地在遍历文件时直接修改
	Languages map[string]*LanguageStats `json:"languages" yaml:"languages"`
}
