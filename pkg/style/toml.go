package style

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	toml "github.com/pelletier/go-toml/v2"
)

// PrintTOML 将任意值以美化并带高亮的方式输出为 TOML 到 writer
//
// 入参规则与 PrintJSON / PrintYAML 类似：
//   - string / []byte: 视为原始 TOML 文本，会尝试解析并重新序列化以规范化输出
//   - 其他任意 Go 值: 使用 toml.Marshal 编码后再渲染
func PrintTOML(w io.Writer, v any) error {
	pretty, err := FormatTOML(v)
	if err != nil {
		return err
	}
	colored := colorizeTOML(pretty)
	_, err = fmt.Fprint(w, colored)
	return err
}

// tokenizeTOMLValue 将 TOML 的值部分进行 token 化并写入 out
func tokenizeTOMLValue(s string, out *strings.Builder, strStyle, numStyle, boolStyle, punctStyle lipgloss.Style) {
	i := 0
	for i < len(s) {
		ch := s[i]
		if ch == '"' || ch == '\'' {
			token, next := readQuotedTokenTOML(s, i)
			out.WriteString(strStyle.Render(token))
			i = next
			continue
		}
		if ch == '[' || ch == ']' || ch == ',' {
			out.WriteString(punctStyle.Render(string(ch)))
			i++
			continue
		}
		if ch == '-' || (ch >= '0' && ch <= '9') {
			j := readNumber(s, i)
			out.WriteString(numStyle.Render(s[i:j]))
			i = j
			continue
		}
		if hasPrefixAt(s, i, "true") {
			out.WriteString(boolStyle.Render("true"))
			i += 4
			continue
		}
		if hasPrefixAt(s, i, "false") {
			out.WriteString(boolStyle.Render("false"))
			i += 5
			continue
		}
		// datetime / offset / local-date-time 简单以 T 或 Z 或 - 为指示，保守处理为普通字符
		if ch == 'T' || ch == 'Z' {
			out.WriteByte(ch)
			i++
			continue
		}
		out.WriteByte(ch)
		i++
	}
}

// readQuotedTokenTOML 与其他文件的读取逻辑相同，返回包含引号的 token 与下一个位置
func readQuotedTokenTOML(s string, i int) (string, int) {
	q := s[i]
	j := i + 1
	escaped := false
	if q == '"' {
		for j < len(s) {
			cj := s[j]
			if cj == '\\' && !escaped {
				escaped = true
				j++
				continue
			}
			if cj == '"' && !escaped {
				j++
				break
			}
			if escaped {
				escaped = false
			}
			j++
		}
		return s[i:j], j
	}
	for j < len(s) {
		cj := s[j]
		if cj == '\'' {
			if j+1 < len(s) && s[j+1] == '\'' {
				j += 2
				continue
			}
			j++
			break
		}
		j++
	}
	return s[i:j], j
}

// PrintTOMLLine 将单行 TOML 以美化样式的方式输出到 writer
func PrintTOMLLine(w io.Writer, v any) error {
	s := v.(string)
	colored := colorizeTOML(s)
	_, err := fmt.Fprint(w, colored)
	return err
}

// FormatTOML 返回美化（规范化）的 TOML 字符串
func FormatTOML(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "\n", nil
	case string:
		src := strings.TrimSpace(x)
		if src == "" {
			return "\n", nil
		}
		var obj any
		if err := toml.Unmarshal([]byte(src), &obj); err != nil {
			return "", err
		}
		b, err := toml.Marshal(obj)
		if err != nil {
			return "", err
		}
		if len(b) == 0 || b[len(b)-1] != '\n' {
			b = append(b, '\n')
		}
		return string(b), nil
	case []byte:
		src := bytes.TrimSpace(x)
		if len(src) == 0 {
			return "\n", nil
		}
		var obj any
		if err := toml.Unmarshal(src, &obj); err != nil {
			return "", err
		}
		b, err := toml.Marshal(obj)
		if err != nil {
			return "", err
		}
		if len(b) == 0 || b[len(b)-1] != '\n' {
			b = append(b, '\n')
		}
		return string(b), nil
	default:
		b, err := toml.Marshal(v)
		if err != nil {
			return "", err
		}
		if len(b) == 0 || b[len(b)-1] != '\n' {
			b = append(b, '\n')
		}
		return string(b), nil
	}
}

// colorizeTOML 对 TOML 文本进行轻量高亮
// 策略：按行处理，识别表头 ([...])、键（= 前的部分）、字符串、数字、布尔、数组和注释
func colorizeTOML(s string) string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorJSONKey).Bold(true)
	strStyle := lipgloss.NewStyle().Foreground(ColorJSONValue)
	numStyle := lipgloss.NewStyle().Foreground(ColorJSONNumber)
	boolStyle := lipgloss.NewStyle().Foreground(ColorJSONBool)
	_ = lipgloss.NewStyle().Foreground(ColorJSONNull)
	punctStyle := lipgloss.NewStyle().Foreground(ColorJSONPunct)
	// comment reuse nullStyle (or punct) — TOML 注释使用暗色
	commentStyle := lipgloss.NewStyle().Foreground(ColorJSONPunct)

	var out strings.Builder
	lines := strings.Split(s, "\n")
	for li, line := range lines {
		if line == "" {
			if li < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}

		trimmed := strings.TrimLeft(line, " \t")
		// 前导缩进保留
		indentLen := len(line) - len(trimmed)
		if indentLen > 0 {
			out.WriteString(line[:indentLen])
		}

		// 注释行
		if strings.HasPrefix(trimmed, "#") {
			out.WriteString(commentStyle.Render(trimmed))
			if li < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}

		// 表头 [table] 或 [[array]]
		if strings.HasPrefix(trimmed, "[[") || strings.HasPrefix(trimmed, "[") {
			// 找到关闭 ]
			idx := strings.Index(trimmed, "]")
			if idx >= 0 {
				out.WriteString(punctStyle.Render(trimmed[:idx+1]))
				rest := trimmed[idx+1:]
				if len(rest) > 0 {
					out.WriteString(rest)
				}
			} else {
				out.WriteString(punctStyle.Render(trimmed))
			}
			if li < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}

		// 行中可能包含注释，先分离
		commentIdx := strings.Index(trimmed, "#")
		content := trimmed
		comment := ""
		if commentIdx >= 0 {
			content = strings.TrimRight(trimmed[:commentIdx], " ")
			comment = trimmed[commentIdx:]
		}

		// 键 = 值
		if eq := indexUnquoted(content, '='); eq >= 0 {
			key := strings.TrimSpace(content[:eq])
			rest := strings.TrimSpace(content[eq+1:])
			out.WriteString(keyStyle.Render(key))
			out.WriteString(" ")
			out.WriteString(punctStyle.Render("="))
			if rest != "" {
				out.WriteString(" ")
				// 对 rest 进行 token 化
				tokenizeTOMLValue(rest, &out, strStyle, numStyle, boolStyle, punctStyle)
			}
		} else {
			// 没有等号，原样输出（可能是继续的多行数组等）
			out.WriteString(content)
		}

		if comment != "" {
			out.WriteByte(' ')
			out.WriteString(commentStyle.Render(comment))
		}

		if li < len(lines)-1 {
			out.WriteByte('\n')
		}
	}

	return out.String()
}
