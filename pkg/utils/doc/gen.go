// Package doc provides utilities for working with documentation.
// 获取 golang 项目的文档字符串，并提供一些工具函数，用于获取渲染样式
package doc

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

var (
	// 包级日志记录器，默认不输出（nop），在 GetDoc 中可被覆盖
	log zerolog.Logger = zerolog.Nop()
)

// GetDoc 返回仓库的文档字符串
// 实现策略:
//
//  1. 解析 options 中的 Root 路径
//  2. 根据 Mode 解析文档
//  3. 如果 Mode 是 Godoc，则提取包文档，如果是 Markdown，则直接读取文件
//
// 如果两者都失败则返回错误
func GetDoc(logger *zerolog.Logger, opt Options, root, path string) (string, error) {
	if logger != nil {
		log = *logger
	} else {
		log = zerolog.Nop()
	}

	// 验证 options
	if err := opt.Validate(); err != nil {
		log.Debug().Err(err).Msg("options validation failed")
		return "", err
	}

	// 根据模式解析文档
	if s, err := resolveMode(opt, root, path); err != nil {
		return "", err
	} else if strings.TrimSpace(s) != "" {
		return s, nil
	}

	return "", fmt.Errorf("no go files found with mode %s under %s", opt.Mode, root)
}

// resolveMode 根据传入的 mode 判断工作模式
//
//  1. Godoc 模式：使用 go/doc 提取文档
//  2. Markdown 模式：直接渲染项目中的 markdown 文件
//
// 返回解析后的文档字符串
func resolveMode(opt Options, root, path string) (string, error) {
	switch opt.Mode {
	case ModeGodoc:
		return GetGoDoc(opt, root, path)
	case ModeMarkdown:
		// Markdown 模式：强制手动传入 markdown 文件用于渲染。
		if path == "" {
			return "", fmt.Errorf("mode %s requires a markdown file path; pass a file or set mode to ModeGodoc for package docs", opt.Mode)
		}
		if s, err := readMarkdownFile(path); err == nil && strings.TrimSpace(s) != "" {
			return s, nil
		}
		return "", fmt.Errorf("markdown file %s is empty or unreadable", path)
	}
	return "", fmt.Errorf("no documentation found with mode %s under %s", opt.Mode, root)
}

// readMarkdownFile 尝试按传入路径读取 markdown 文件，支持相对 root 的路径或绝对路径
func readMarkdownFile(file string) (string, error) {
	// 读取文件内容
	b, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	// Debug: read markdown file
	log.Debug().Str("file", file).Int("bytes", len(b)).Msg("readMarkdownFile: read file")
	return string(b), nil
}
