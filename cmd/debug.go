package cmd

import (
	"github.com/spf13/cobra"
)

var (
	debugCmd = &cobra.Command{
		Use:   "debug",
		Short: "Debug related commands",
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
	)
}
