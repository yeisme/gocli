package project

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/yeisme/gocli/pkg/style"
)

// FmtOptions 是用于格式化代码的选项
type FmtOptions struct {
	List    bool   // 列出可用的格式化器
	Path    string // 指定格式化路径，默认为当前目录
	Verbose bool   // 逐行输出结果

	ConfigPath string // 配置文件路径
}

// RunFmt 执行代码格式化操作（使用 golangci-lint fmt）
// 行为：
//
//	List=true  -> golangci-lint formatters
//	List=false -> golangci-lint fmt <path>
//
// 返回完整输出，同时在 Verbose 模式下逐行通过 logger 打印
func RunFmt(options FmtOptions, out io.Writer) error {
	var args []string
	if options.List {
		args = append(args, "formatters") // golangci-lint formatters
	} else {
		target := options.Path
		if target == "" {
			target = "."
		}
		if options.ConfigPath != "" {
			args = append(args, "fmt", "--config", options.ConfigPath)
		}
		args = append(args, "fmt", target) // golangci-lint fmt <path>

	}

	var output string
	var err error
	if options.List {
		output, err = execGolangCILint(args, nil, nil)
	} else {
		var stdout, stderr io.Writer
		if out != nil {
			stdout = out
			stderr = out
		} else {
			var discard bytes.Buffer
			stdout = &discard
			stderr = &discard
		}
		_, err = execGolangCILint(args, stdout, stderr)
	}
	if err != nil {
		return err
	}

	if options.List {
		// 解析并美化打印格式化器列表
		formatters := parseFormatterOutput(output)
		// 分组：已启用 / 未启用
		var enabled, disabled []style.Formatter
		for _, f := range formatters {
			if f.Enabled {
				enabled = append(enabled, f)
			} else {
				disabled = append(disabled, f)
			}
		}
		_, _ = fmt.Fprintln(out)
		_ = style.PrintHeading(out, "Enabled Formatters")
		_ = style.PrintFormatterList(out, enabled)
		_, _ = fmt.Fprintln(out)
		_ = style.PrintHeading(out, "Disabled Formatters")
		_ = style.PrintFormatterList(out, disabled)
	} else if options.Verbose && output != "" {
		// 逐行输出结果
		scanner := bufio.NewScanner(strings.NewReader(output))
		for scanner.Scan() {
			line := scanner.Text()
			log.Info().Msg(line)
		}
	}
	return nil
}

var formatterLineRE = regexp.MustCompile(`^([a-zA-Z0-9_]+):\s+(.*)$`)

// parseFormatterOutput 解析 golangci-lint formatters 命令输出
// 样例：
// Enabled by your configuration formatters:
//
//	gofmt: Check if the code is formatted according to 'gofmt' command.
//
// Disabled by your configuration formatters:
//
//	gci: Check ...
func parseFormatterOutput(out string) []style.Formatter {
	var list []style.Formatter
	enabledSection := false
	disabledSection := false
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "Enabled by"):
			enabledSection = true
			disabledSection = false
			continue
		case strings.HasPrefix(line, "Disabled by"):
			enabledSection = false
			disabledSection = true
			continue
		}
		if m := formatterLineRE.FindStringSubmatch(line); len(m) == 3 {
			list = append(list, style.Formatter{
				Name:        m[1],
				Description: m[2],
				Enabled:     enabledSection && !disabledSection,
			})
		}
	}
	return list
}
