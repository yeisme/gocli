package count

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yeisme/gocli/pkg/models"
)

// SingleFileCounter 是 CountFile 的一个基础实现
// 采用简单的基于扩展名的语言识别与注释统计。
type SingleFileCounter struct{}

// CountSingleFile 通过组合 CountLines/CountBlanks/CountComments 汇总单文件信息
func (s *SingleFileCounter) CountSingleFile(ctx context.Context, filePath string, opts Options) (*models.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 可选：大小限制（超限直接返回错误，交由上层跳过）
	if opts.MaxFileSizeBytes > 0 {
		fi, err := os.Stat(filePath)
		if err == nil && fi.Size() > opts.MaxFileSizeBytes {
			return nil, fmt.Errorf("file size exceeds limit: %d > %d", fi.Size(), opts.MaxFileSizeBytes)
		}
	}

	total, err := s.CountLines(ctx, filePath)
	if err != nil {
		return nil, err
	}
	blanks, err := s.CountBlanks(ctx, filePath)
	if err != nil {
		return nil, err
	}
	comments, err := s.CountComments(ctx, filePath)
	if err != nil {
		return nil, err
	}

	code := max(total-blanks-comments, 0)

	info := &models.FileInfo{
		Path:     filePath,
		Language: detectLanguage(filePath),
		Stats: models.Stats{
			Code:     code,
			Comments: comments,
			Blanks:   blanks,
		},
	}

	// 语言特定信息可在其它实现中填充（例如实现 CountGoFile）
	if !opts.WithLanguageSpecific {
		info.LanguageSpecific = nil
	}

	return info, nil
}

// CountLines 统计总行数（以换行分割）
func (s *SingleFileCounter) CountLines(ctx context.Context, filePath string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	// 提高最大 token 大小，避免超长行导致扫描失败
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024) // 1MB 单行
	count := 0
	for sc.Scan() {
		count++
		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

// CountBlanks 统计空白行（仅空白字符）
func (s *SingleFileCounter) CountBlanks(ctx context.Context, filePath string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	blanks := 0
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		line := sc.Text()
		if isBlank(line) {
			blanks++
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	return blanks, nil
}

// CountComments 统计注释行（基于语言的简易规则）
func (s *SingleFileCounter) CountComments(ctx context.Context, filePath string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	style := getCommentStyle(strings.ToLower(filepath.Ext(filePath)))
	if style.isNone() {
		return 0, nil
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	comments := 0
	inBlock := false
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		if inBlock {
			comments++
			if style.blockEnd != "" && strings.Contains(line, style.blockEnd) {
				inBlock = false
			}
			continue
		}

		// 单行注释（要求行首/去空白后以注释开头，避免误判 URL 等）
		if hasSingleLineCommentPrefix(line, style.single) {
			comments++
			continue
		}

		// 块注释开始
		if style.blockStart != "" && strings.Contains(line, style.blockStart) {
			comments++ // 该行计作注释
			if style.blockEnd != "" && !strings.Contains(line, style.blockEnd) {
				inBlock = true
			}
			continue
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	// 若文件结束仍在块注释中，不再额外处理
	return comments, nil
}

func isBlank(s string) bool {
	for _, r := range s {
		if !isSpace(r) {
			return false
		}
	}
	return true
}

func isSpace(r rune) bool {
	// 与 unicode.IsSpace 类似，显式处理常见空白
	switch r {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	default:
		// 兜底：处理其它 unicode 空白
		return strings.ContainsRune("\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000", r)
	}
}

func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	if lang, ok := ExtToLang[ext]; ok {
		return lang
	}
	return "Unknown"
}

type commentStyle struct {
	single     []string
	blockStart string
	blockEnd   string
}

func (c commentStyle) isNone() bool {
	return len(c.single) == 0 && c.blockStart == "" && c.blockEnd == ""
}

func getCommentStyle(ext string) commentStyle {
	if style, ok := LangToComment[ExtToLang[ext]]; ok {
		return style
	}
	return commentStyle{}
}

func hasSingleLineCommentPrefix(line string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return false
	}
	for _, p := range prefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

var _ File = (*SingleFileCounter)(nil)
