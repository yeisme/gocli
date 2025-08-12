package project

import (
	"bufio"
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
}

// RunLint 执行 lint 操作
func RunLint(options LintOptions, out io.Writer) error {
	var args []string
	if options.List {
		args = append(args, "linters") // golangci-lint linters
	} else if options.Fix {
		args = append(args, "run", "--fix") // golangci-lint run --fix
	} else {
		args = append(args, "run") // golangci-lint run
	}
	output, err := execGolangCILint(args)
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
		_, _ = fmt.Fprintln(out)
		_ = style.PrintHeading(out, "Enabled Linters")
		_ = style.PrintFormatterList(out, enabled)
		_, _ = fmt.Fprintln(out)
		_ = style.PrintHeading(out, "Disabled Linters")
		_ = style.PrintFormatterList(out, disabled)
	} else if options.Verbose {
		scanner := bufio.NewScanner(strings.NewReader(output))
		for scanner.Scan() {
			log.Info().Msg(scanner.Text())
		}
	}
	return nil
}

func execGolangCILint(args []string) (string, error) {
	output, err := tools.NewExecutor("golangci-lint", args...).Output()
	if err != nil {
		log.Error().Err(err).Msg("failed to execute golangci-lint")
		return "", err
	}
	return output, nil
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
