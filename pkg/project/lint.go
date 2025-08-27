package project

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/tools"
)

// LintOptions 是用于 lint 代码的选项
type LintOptions struct {
	List    bool
	Fix     bool
	Verbose bool

	Config struct {
		Validate bool
		Path     bool
	}
	ConfigPath string // 配置文件路径

}

// RunLint 执行 lint 操作
func RunLint(options LintOptions, out io.Writer) error {
	var args []string
	if options.List {
		args = append(args, "linters") // golangci-lint linters
	} else if options.Fix {
		args = append(args, "run", "--fix") // golangci-lint run --fix
	} else if options.Config.Validate {
		// golangci-lint config validate [-c path]
		args = append(args, "config", "validate")
	} else if options.Config.Path {
		// golangci-lint config path [-c path]
		args = append(args, "config", "path")
	} else if options.ConfigPath != "" {
		args = append(args, "run", "--config", options.ConfigPath)
	} else {
		args = append(args, "run")
	}

	// 如果是 config 子命令，且提供了 ConfigPath，则通过 -c 传入，而不是作为位置参数
	if (options.Config.Validate || options.Config.Path) && options.ConfigPath != "" {
		args = append(args, "-c", options.ConfigPath)
	}

	var output string
	var err error

	// list 模式需要解析输出，因此捕获到字符串；
	// 其他模式直接把 stdout/stderr 写到 out（例如 run --fix）
	if options.List {
		output, err = execGolangCILint(args, nil, nil)
	} else {
		// 允许 out 为 nil 的情况
		var stderr io.Writer
		var stdout io.Writer
		if out != nil {
			stdout = out
			// 将 stderr 也输出到同一 writer，便于用户看到错误
			stderr = out
		} else {
			// 虽然 unlikely，但保持健壮性：如果 out 为空，把输出丢弃
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
		linters := parseLintersOutput(output)
		var enabled, disabled []style.Formatter
		for _, l := range linters {
			if l.Enabled {
				enabled = append(enabled, l)
			} else {
				disabled = append(disabled, l)
			}
		}
		fmt.Fprintln(out)
		_ = style.PrintHeading(out, "Enabled Linters")
		_ = style.PrintFormatterList(out, enabled)
		fmt.Fprintln(out)
		_ = style.PrintHeading(out, "Disabled Linters")
		_ = style.PrintFormatterList(out, disabled)
	} else if options.Verbose && output != "" {
		scanner := bufio.NewScanner(strings.NewReader(output))
		for scanner.Scan() {
			log.Info().Msg(scanner.Text())
		}
	}
	return nil
}

// execGolangCILint 封装对 golangci-lint 的调用：
//   - 当 stdout/stderr 为 nil 时，使用 Output 捕获并返回 stdout 字符串；
//   - 当提供 stdout/stderr 时，使用 RunStreaming 直接写入并返回空字符串
func execGolangCILint(args []string, stdout, stderr io.Writer) (string, error) {
	_, err := tools.TestExists("golangci-lint")
	if err != nil {
		return "", err
	}

	exec := tools.NewExecutor("golangci-lint", args...)
	if stdout == nil && stderr == nil {
		output, err := exec.Output()
		if err != nil {
			return "", err
		}
		return output, nil
	}
	if err := exec.RunStreaming(stdout, stderr); err != nil {
		return "", err
	}
	return "", nil
}

var linterLineRE = regexp.MustCompile(`^([a-zA-Z0-9_-]+):\s+(.*)$`)

// parseLintersOutput 解析 golangci-lint linters 输出，返回格式化后的结构
func parseLintersOutput(out string) []style.Formatter {
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
		if m := linterLineRE.FindStringSubmatch(line); len(m) == 3 {
			list = append(list, style.Formatter{
				Name:        m[1],
				Description: m[2],
				Enabled:     enabledSection && !disabledSection,
			})
		}
	}
	return list
}
