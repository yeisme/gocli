package cmd

import (
	"github.com/spf13/cobra"
	debugPkg "github.com/yeisme/gocli/pkg/debug"
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
		Short: "Execute tracing for debugging",
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
			return debugPkg.PrintVersionTable(cmd.OutOrStdout(), exe)
		},
		Args: cobra.ExactArgs(1),
	}
)

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
}
