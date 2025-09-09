package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/debug"
)

var (
	debugCmd = &cobra.Command{
		Use:     "debug",
		Short:   "Debug related commands",
		Aliases: []string{"dbg", "d"},
	}

	debugExeInfoCmd = &cobra.Command{
		Use:   "exeinfo",
		Short: "Show executable information",
	}

	debugCoreCmd = &cobra.Command{
		Use:   "file",
		Short: "Debug Core Dump files",
	}

	debugTraceCmd = &cobra.Command{
		Use:   "trace",
		Short: "View or analyze Go execution trace (wrapper of 'go tool trace')",
		Long: `Run and view Go execution trace.

Examples:
  # Run trace server on default address and open in browser manually
  gocli debug trace trace.out

  # Specify HTTP address
  gocli debug trace --http :6061 trace.out

  # Generate pprof-like report from trace
  gocli debug trace --pprof sched trace.out

  # Provide test binary (rarely needed for Go >1.7)
  gocli debug trace ./pkg.test trace.out

Wrapper logic:
  gocli will map the provided flags to 'go tool trace'.
  Argument rules:
    1 arg  -> trace file
    2 args -> binary + trace file (kept for backward compatibility)
`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse positional args
			var (
				bin       string
				traceFile string
			)
			if len(args) == 1 {
				traceFile = args[0]
			} else {
				bin = args[0]
				traceFile = args[1]
			}

			// Validate files
			if err := ensureFile(traceFile); err != nil {
				return fmt.Errorf("trace file invalid: %w", err)
			}
			if bin != "" {
				if err := ensureFile(bin); err != nil {
					return fmt.Errorf("binary invalid: %w", err)
				}
			}

			// Collect options
			opt := debug.TraceOptions{
				HTTPAddr: traceHTTPAddr,
				PProf:    tracePProfType,
				Debug:    traceDebugMode,
				Verbose:  traceVerbose,
			}

			return debug.RunTrace(cmd.ErrOrStderr(), cmd.OutOrStdout(), opt, bin, traceFile)
		},
	}

	debugProfileCmd = &cobra.Command{
		Use:   "profile",
		Short: "Profile application performance",
	}

	debugStackCmd = &cobra.Command{
		Use:   "stack",
		Short: "Show stack trace information",
	}

	debugMemCmd = &cobra.Command{
		Use:   "mem",
		Short: "Show memory statistics",
	}

	debugGoroutineCmd = &cobra.Command{
		Use:   "goroutines",
		Short: "Show goroutines count and optionally full dump",
	}

	debugDumpCmd = &cobra.Command{
		Use:   "dump",
		Short: "Create diagnostic dumps (heap/goroutine)",
	}

	debugBinInfoCmd = &cobra.Command{
		Use:   "bininfo",
		Short: "Show binary build info (module, vcs, etc)",
	}

	debugVersionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show Go version information",
		Long: `
Show Go version information. If an executable is provided, it will display the Go version used to build that executable.

Usage:
  gocli debug version [executable_path]

Examples:
  gocli debug version /path/to/your/executable
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var exe string
			if len(args) > 0 {
				exe = args[0]
			}
			return debug.PrintVersionTable(cmd.OutOrStdout(), exe)
		},
		Args: cobra.ExactArgs(1),
	}

	// trace flags (bound in init)
	traceHTTPAddr  string
	tracePProfType string
	traceDebugMode string
	traceVerbose   bool
)

// ensureFile checks existence & regular file
func ensureFile(path string) error {
	if path == "" {
		return errors.New("empty path")
	}
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}

// registerTraceFlags binds flags for the trace command
func registerTraceFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&traceHTTPAddr, "http", ":0", "HTTP server address for trace viewer (passed to go tool trace -http)")
	cmd.Flags().StringVar(&tracePProfType, "pprof", "", "Generate pprof-like profile (net|sync|syscall|sched)")
	cmd.Flags().StringVar(&traceDebugMode, "d", "", "Print debug info and exit (wire|parsed|footprint)")
	cmd.Flags().BoolVarP(&traceVerbose, "verbose", "v", false, "Show underlying 'go tool trace' command")
}

func init() {
	rootCmd.AddCommand(debugCmd)

	debugCmd.AddCommand(
		debugExeInfoCmd,
		debugCoreCmd,
		debugTraceCmd,
		debugProfileCmd,
		debugStackCmd,
		debugMemCmd,
		debugGoroutineCmd,
		debugDumpCmd,
		debugBinInfoCmd,
		debugVersionCmd,
	)

	// trace flags
	registerTraceFlags(debugTraceCmd)
}
