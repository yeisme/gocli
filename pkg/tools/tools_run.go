package tools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExecuteToolRun finds and executes a tool by name or path. This is an exported
// wrapper so external binaries (like the `gox` shim) can reuse the same logic
// as the main `gocli tools run` implementation in cmd.
func ExecuteToolRun(args []string, out io.Writer, verbose bool, gocliToolsPath string) error {
	// 当无参数时，展示工具列表
	if len(args) == 0 {
		tools := FindTools(verbose, gocliToolsPath)
		if err := PrintToolsTable(out, tools, verbose); err != nil {
			return fmt.Errorf("failed to print tools list in table format: %w", err)
		}
		return nil
	}

	name := args[0]

	// 1) 在已发现的工具中查找（大小写不敏感）
	execPath := ""
	toolsList := FindTools(verbose, gocliToolsPath)
	for i := range toolsList {
		t := toolsList[i]
		if strings.EqualFold(t.Name, name) || strings.EqualFold(filepath.Base(t.Path), name) {
			execPath = t.Path
			break
		}
	}

	// 2) 若输入看起来像路径，则直接使用（包含 Windows 驱动器/分隔符或绝对路径）
	if execPath == "" {
		if strings.ContainsAny(name, ":/\\") || filepath.IsAbs(name) {
			if _, err := os.Stat(name); err == nil {
				execPath = name
			}
		}
	}

	if execPath == "" {
		return fmt.Errorf("tool not found: %s", name)
	}

	// 恢复原始命令行中 run 之后的参数（优先使用未解析的 os.Args）
	raw := rawArgsAfterRun(args)
	execArgs := []string{}
	if len(raw) > 1 {
		execArgs = raw[1:]
	}

	exec := NewExecutor(execPath, execArgs...)
	if err := exec.RunStreaming(os.Stdout, os.Stderr); err != nil {
		if ee, ok := err.(*ExecError); ok {
			return fmt.Errorf("tool %s failed: exit=%d stderr=%s", execPath, ee.ExitCode(), ee.CleanStderr())
		}
		return err
	}
	return nil
}

// rawArgsAfterRun tries to reconstruct the raw argv slice starting at the
// tool name. It prefers the original os.Args (so flags intended for the
// executed tool are preserved), and falls back to the cobra-parsed args.
func rawArgsAfterRun(cobraArgs []string) []string {
	if len(cobraArgs) == 0 {
		return cobraArgs
	}

	toolName := cobraArgs[0]
	argv := os.Args
	for i := range argv {
		if argv[i] == toolName {
			if i > 0 && (argv[i-1] == "run" || argv[i-1] == "x" || argv[i-1] == "exec") {
				return argv[i:]
			}
			if i > 1 && (argv[i-2] == "run" || argv[i-2] == "x" || argv[i-2] == "exec") {
				return argv[i:]
			}
		}
	}

	// 回退到 cobra 提供的 args
	return cobraArgs
}
