package style

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// PrintYAML 将任意值以美化并带高亮的方式输出为 YAML 到 writer
//
// 入参规则与 PrintJSON 类似：
//   - string / []byte: 视为原始 YAML 文本，会尝试解析并重新序列化以规范化输出
//   - 其他任意 Go 值: 使用 yaml.Marshal 编码后再渲染
func PrintYAML(w io.Writer, v any) error {
	pretty, err := FormatYAML(v)
	if err != nil {
		return err
	}
	colored := colorizeYAML(pretty)
	_, err = fmt.Fprint(w, colored)
	return err
}

// PrintYAMLLine 将单行 YAML 以美化样式的方式输出到 writer
func PrintYAMLLine(w io.Writer, v any) error {
	s := v.(string)
	colored := colorizeYAML(s)
	_, err := fmt.Fprint(w, colored)
	return err
}

// FormatYAML 返回美化（规范化缩进）的 YAML 字符串
func FormatYAML(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "null\n", nil
	case string:
		// 尝试解析后再序列化以规范化输出
		src := strings.TrimSpace(x)
		if src == "" {
			return "null\n", nil
		}
		var obj any
		if err := yaml.Unmarshal([]byte(src), &obj); err != nil {
			return "", err
		}
		b, err := yaml.Marshal(obj)
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
			return "null\n", nil
		}
		var obj any
		if err := yaml.Unmarshal(src, &obj); err != nil {
			return "", err
		}
		b, err := yaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		if len(b) == 0 || b[len(b)-1] != '\n' {
			b = append(b, '\n')
		}
		return string(b), nil
	default:
		b, err := yaml.Marshal(v)
		if err != nil {
			return "", err
		}
		if len(b) == 0 || b[len(b)-1] != '\n' {
			b = append(b, '\n')
		}
		return string(b), nil
	}
}

// colorizeYAML 对 YAML 文本进行轻量高亮（键名、标点、数字、布尔、null）
// 实现策略：按行处理，识别未被引号包裹的键（行首或缩进后直到第一个冒号），
// 对冒号、连字号（-）等标记使用 punctStyle，并对数值/布尔/null 使用已有样式。
func colorizeYAML(s string) string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorJSONKey).Bold(true)
	strStyle := lipgloss.NewStyle().Foreground(ColorJSONValue)
	numStyle := lipgloss.NewStyle().Foreground(ColorJSONNumber)
	boolStyle := lipgloss.NewStyle().Foreground(ColorJSONBool)
	nullStyle := lipgloss.NewStyle().Foreground(ColorJSONNull)
	punctStyle := lipgloss.NewStyle().Foreground(ColorJSONPunct)

	var out strings.Builder
	lines := strings.Split(s, "\n")
	for li, line := range lines {
		if line == "" {
			if li < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}
		trimmed := line
		// detect leading indent
		indentLen := 0
		for i := 0; i < len(line); i++ {
			if line[i] == ' ' || line[i] == '\t' {
				indentLen++
			} else {
				break
			}
		}
		if indentLen > 0 {
			out.WriteString(line[:indentLen])
			trimmed = line[indentLen:]
		}

		// list item marker
		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			out.WriteString(punctStyle.Render("-"))
			if len(trimmed) > 1 {
				out.WriteByte(' ')
				trimmed = strings.TrimPrefix(trimmed, "- ")
			} else {
				trimmed = ""
			}
		}

		if trimmed == "" {
			if li < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}

		// 找到第一个未被引号包裹的冒号，作为键分隔
		idx := indexUnquoted(trimmed, ':')
		if idx > 0 {
			key := trimmed[:idx]
			rest := trimmed[idx+1:]
			out.WriteString(keyStyle.Render(key))
			out.WriteString(punctStyle.Render(":"))
			// rest 可能以空格开始
			if len(rest) > 0 && rest[0] == ' ' {
				out.WriteByte(' ')
				rest = rest[1:]
			}
			// 对 rest 进行 token 化
			tokenizeLine(rest, &out, strStyle, numStyle, boolStyle, nullStyle)
		} else {
			// 没有冒号，整行当作值或注释，尝试以 token 化处理（例如纯数字、布尔）
			tokenizeLine(trimmed, &out, strStyle, numStyle, boolStyle, nullStyle)
		}

		if li < len(lines)-1 {
			out.WriteByte('\n')
		}
	}

	return out.String()
}

// indexUnquoted 在行中找到第一个不在引号中的目标字符位置，找不到返回 -1
func indexUnquoted(line string, target byte) int {
	inQuote := rune(0)
	escaped := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if inQuote != 0 {
			if inQuote == '"' {
				if ch == '\\' && !escaped {
					escaped = true
					continue
				}
				if ch == '"' && !escaped {
					inQuote = 0
				}
				if escaped {
					escaped = false
				}
				continue
			}
			// single quoted: a pair of single quotes represents an escaped quote
			if inQuote == '\'' {
				if ch == '\'' {
					if i+1 < len(line) && line[i+1] == '\'' {
						// skip the doubled quote
						i++
						continue
					}
					inQuote = 0
				}
				continue
			}
		} else {
			if ch == '"' {
				inQuote = '"'
				escaped = false
				continue
			}
			if ch == '\'' {
				inQuote = '\''
				continue
			}
			if ch == target {
				return i
			}
		}
	}
	return -1
}

// tokenizeLine 将一行（不含键和冒号）进行 token 化并写入 out
func tokenizeLine(s string, out *strings.Builder, strStyle, numStyle, boolStyle, nullStyle lipgloss.Style) {
	i := 0
	for i < len(s) {
		ch := s[i]
		if ch == '"' || ch == '\'' {
			token, next := readQuotedToken(s, i)
			out.WriteString(strStyle.Render(token))
			i = next
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
		if hasPrefixAt(s, i, "null") || s[i] == '~' {
			if s[i] == '~' {
				out.WriteString(nullStyle.Render("~"))
				i++
			} else {
				out.WriteString(nullStyle.Render("null"))
				i += 4
			}
			continue
		}
		out.WriteByte(ch)
		i++
	}
}

// readQuotedToken 从位置 i 开始读取包含引号的 token，返回 token 字符串和下一个位置
func readQuotedToken(s string, i int) (string, int) {
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
	// single quoted: two consecutive '' 表示转义
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
