package tools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/utils/executor"
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

	exec := executor.NewExecutor(execPath, execArgs...)
	if err := exec.RunStreaming(os.Stdout, os.Stderr); err != nil {
		if ee, ok := err.(*executor.ExecError); ok {
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

// SetRunHelpFunc configures a help printer for "run"-style commands that
// should hide inherited/global flags and only display usage, long text,
// local flags and available subcommands. This is used by both the
// `gocli tools run` command and the `gox` shim so help output is identical.
func SetRunHelpFunc(cmd *cobra.Command) {
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		out := cmd.OutOrStdout()
		// Usage
		if cmd.Runnable() {
			fmt.Fprintf(out, "Usage: %s\n\n", cmd.UseLine())
		}
		// Long description
		if cmd.Long != "" {
			fmt.Fprintln(out, strings.TrimSpace(cmd.Long))
			fmt.Fprintln(out)
		}
		// Local flags only (omits inherited/global flags)
		if cmd.HasAvailableLocalFlags() {
			fmt.Fprintln(out, "Flags:")
			fmt.Fprint(out, cmd.LocalFlags().FlagUsages())
			fmt.Fprintln(out)
		}
		// Subcommands
		if cmd.HasAvailableSubCommands() {
			fmt.Fprintln(out, "Available Commands:")
			for _, c := range cmd.Commands() {
				if c.IsAvailableCommand() {
					fmt.Fprintf(out, "  %s\t%s\n", c.Name(), c.Short)
				}
			}
			fmt.Fprintln(out)
		}
	})
}

// ShowRunHelpIfRequested inspects the original os.Args to determine if the
// user asked for help specifically for a run-style command (e.g. `gocli run
// --help` or `gox --help`) and prints help then exits. Prefer calling this in
// a command's PreRun to preserve flags intended for the executed tool.
func ShowRunHelpIfRequested(cmd *cobra.Command) {
	argv := os.Args
	for i := range argv {
		if argv[i] == "run" || argv[i] == "x" || argv[i] == "exec" {
			if i+1 < len(argv) {
				next := argv[i+1]
				if next == "--help" || next == "-h" {
					_ = cmd.Help()
					os.Exit(0)
				}
			}
			break
		} else if argv[i] == "--help" || argv[i] == "-h" {
			_ = cmd.Help()
			os.Exit(0)
		}
	}
}

// ToolsRunMsg 用于显示 gocli tools run 的帮助信息
var ToolsRunMsg = struct {
	Short string
	Long  string
}{
	Short: "Run a tool",
	Long: `
Run a tool by configured name or by explicit path and forward all remaining
arguments to the executed binary unchanged.

Basic usage:
  gocli tool run <tool> [args...]
  gox run <tool> [args...]

Examples:
  # Run a configured tool named "task"
  gocli tools run task
  gox run task

  # Execute a binary by path and forward flags/args
  gocli tools run task --list
  gox run task --list

Notes:
  - Use 'gocli tools list' to inspect available configured tools and their
    install paths.
  - If the first argument matches a configured tool name (case-insensitive),
    gocli will run the discovered binary for that tool.
  - If the argument looks like a path (contains path separators or is absolute),
    and the file exists, gocli will execute that path directly.
  - All flags and arguments after the tool name are forwarded verbatim to the
    invoked executable. Unknown flags are allowed so flags intended for the
    executed tool are not interpreted by cobra.
`,
}
