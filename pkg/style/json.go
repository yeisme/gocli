package style

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// PrintJSON 将任意值以美化（缩进）并带有简洁高亮的方式输出到 writer
//
// 入参支持:
//   - string / []byte: 视为原始 JSON 文本；会尝试校验并缩进
//   - 其他任意 Go 值: 使用 [json.MarshalIndent] 编码后再渲染
//
// 高亮策略（与主题色保持一致）:
//   - 键名: AccentPrimary + Bold
//   - 标点（{}[],:）: Border 色
//   - 数字 / 布尔 / null: AccentPrimary
//   - 字符串值与空白: 默认文本色
func PrintJSON(w io.Writer, v any) error {
	pretty, err := FormatJSON(v)
	if err != nil {
		return err
	}
	colored := colorizeJSON(pretty)
	_, err = fmt.Fprint(w, colored)
	return err
}

// PrintJSONLine 将单行 JSON 以美化样式的方式输出到 writer
func PrintJSONLine(w io.Writer, v any) error {
	s := v.(string)
	colored := colorizeJSON(s)
	_, err := fmt.Fprint(w, colored)
	return err
}

// FormatJSON 返回美化（缩进）的 JSON 字符串
// 参见 PrintJSON 的入参规则
func FormatJSON(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "null\n", nil
	case string:
		return indentJSON([]byte(x))
	case []byte:
		return indentJSON(x)
	default:
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return "", err
		}
		// MarshalIndent 不会自动补换行，保持与 indentJSON 一致性，这里附加一个换行
		if len(b) == 0 || b[len(b)-1] != '\n' {
			b = append(b, '\n')
		}
		return string(b), nil
	}
}

// indentJSON 校验并缩进原始 JSON 字节
func indentJSON(src []byte) (string, error) {
	// 允许输入存在多余空白
	src = bytes.TrimSpace(src)
	if len(src) == 0 {
		return "null\n", nil
	}
	var out bytes.Buffer
	if err := json.Indent(&out, src, "", "  "); err != nil {
		return "", err
	}
	if out.Len() == 0 || out.Bytes()[out.Len()-1] != '\n' {
		_ = out.WriteByte('\n')
	}
	return out.String(), nil
}

// colorizeJSON 对已经缩进好的 JSON 文本进行轻量高亮
// 注意: 只对 JSON 语义 token 着色；缩进与空白保持原样
func colorizeJSON(s string) string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorJSONKey).Bold(true)
	strStyle := lipgloss.NewStyle().Foreground(ColorJSONValue)
	numStyle := lipgloss.NewStyle().Foreground(ColorJSONNumber)
	boolStyle := lipgloss.NewStyle().Foreground(ColorJSONBool)
	nullStyle := lipgloss.NewStyle().Foreground(ColorJSONNull)
	punctStyle := lipgloss.NewStyle().Foreground(ColorJSONPunct)

	var b bytes.Buffer
	inString := false
	// 缓存当前字符串 token（包含起始及结束引号），结束时一次性渲染
	var sb bytes.Buffer

	i := 0
	for i < len(s) {
		ch := s[i]
		if inString {
			sb.WriteByte(ch)
			if ch == '"' && !isEscaped(sb.Bytes()) {
				// 字符串结束，判断是否为 key: 向前看下一个非空白字符是否为 ':'
				inString = false
				isKey := false
				// 在源串 s 中，从当前位置 i+1 向后寻找第一个非空白字符
				j := i + 1
				for j < len(s) && unicode.IsSpace(rune(s[j])) {
					j++
				}
				if j < len(s) && s[j] == ':' {
					isKey = true
				}
				token := sb.String()
				if isKey {
					b.WriteString(keyStyle.Render(token))
				} else {
					b.WriteString(strStyle.Render(token))
				}
				sb.Reset()
			}
			i++
			continue
		}

		switch ch {
		case '"':
			inString = true
			sb.WriteByte(ch)
			i++
			continue
		case '{', '}', '[', ']', ':', ',':
			b.WriteString(punctStyle.Render(string(ch)))
			i++
			continue
		default:
			// 识别数字
			if ch == '-' || (ch >= '0' && ch <= '9') {
				j := readNumber(s, i)
				b.WriteString(numStyle.Render(s[i:j]))
				i = j
				continue
			}
			// 识别 true/false/null
			if hasPrefixAt(s, i, "true") {
				b.WriteString(boolStyle.Render("true"))
				i += 4
				continue
			}
			if hasPrefixAt(s, i, "false") {
				b.WriteString(boolStyle.Render("false"))
				i += 5
				continue
			}
			if hasPrefixAt(s, i, "null") {
				b.WriteString(nullStyle.Render("null"))
				i += 4
				continue
			}
			// 其他（空白等）原样输出
			b.WriteByte(ch)
			i++
		}
	}

	// 防御: 若字符串未闭合（不太可能，因为输入已是有效 JSON），将缓存回退为原样
	if sb.Len() > 0 {
		if inString {
			b.WriteString(sb.String())
		}
	}
	return b.String()
}

// isEscaped 判断当前字符串缓冲区结尾的引号是否被反斜杠转义
func isEscaped(buf []byte) bool {
	if len(buf) < 2 || buf[len(buf)-1] != '"' {
		return false
	}
	// 统计倒数第二个开始连续的反斜杠个数
	cnt := 0
	for i := len(buf) - 2; i >= 0 && buf[i] == '\\'; i-- {
		cnt++
	}
	return cnt%2 == 1
}

// readNumber 返回从 i 开始的数字 token 的结束位置（半开区间）
func readNumber(s string, i int) int {
	j := i
	// 负号
	if j < len(s) && s[j] == '-' {
		j++
	}
	// 整数部分
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	// 小数
	if j < len(s) && s[j] == '.' {
		j++
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
	}
	// 指数
	if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
		j++
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
	}
	return j
}

func hasPrefixAt(s string, i int, pref string) bool {
	if i+len(pref) > len(s) {
		return false
	}
	if s[i:i+len(pref)] != pref {
		return false
	}
	// 边界控制: 前后需是分隔（空白、标点）或边界
	// 前一位（若存在）不可为字母/数字/下划线
	if i > 0 {
		r := rune(s[i-1])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return false
		}
	}
	if i+len(pref) < len(s) {
		r := rune(s[i+len(pref)])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return false
		}
	}
	return true
}
