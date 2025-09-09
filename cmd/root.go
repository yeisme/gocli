package cmd

import (
	"fmt"
	"os"
	"runtime/pprof"
	"runtime/trace"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/context"
	log2 "github.com/yeisme/gocli/pkg/utils/log"
	"github.com/yeisme/gocli/pkg/utils/version"
)

var (
	gocliCtx *context.GocliContext
	log      log2.Logger

	// Global flags
	globalFlags       = context.GlobalFlags{}
	configPathFlag    = globalFlags.ConfigPath
	debugFlag         = globalFlags.Debug
	verboseFlag       = globalFlags.Verbose
	quietFlag         = globalFlags.Quiet
	cpuProfileFlag    = globalFlags.CPUProfile
	traceFlag         = globalFlags.Trace
	versionEnableFlag = globalFlags.VersionEnable
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gocli",
	Short: "gocli is a CLI application for managing your Go projects",
	Long:  `gocli is a command line interface application that helps you manage your Go projects efficiently.`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionEnableFlag {
			fmt.Fprintln(cmd.OutOrStdout(), version.GetShortVersionString())
			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No arguments provided")
			_ = cmd.Help()
		}
	},
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		if cpuProfileFlag != "" {
			f, err := os.Create(cpuProfileFlag)
			if err != nil {
				log.Fatal().Err(err).Msg("could not create CPU profile")
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				log.Fatal().Err(err).Msg("could not start CPU profile")
			}
		}
		if traceFlag != "" {
			f, err := os.Create(traceFlag)
			if err != nil {
				log.Fatal().Err(err).Msg("could not create trace file")
			}
			if err := trace.Start(f); err != nil {
				log.Fatal().Err(err).Msg("could not start trace")
			}
		}
		ctx := context.InitGocliContext(configPathFlag, debugFlag, verboseFlag, quietFlag)

		gocliCtx = ctx
		log = ctx.Logger

		log.Info().Msgf("Execute Command: %s %s", "gocli", strings.Join(os.Args[1:], " "))
	},
	PersistentPostRun: func(_ *cobra.Command, _ []string) {
		if cpuProfileFlag != "" {
			pprof.StopCPUProfile()
		}
		if traceFlag != "" {
			trace.Stop()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPathFlag, "config", "c", "", "config file")
	rootCmd.PersistentFlags().StringVar(&cpuProfileFlag, "cpu-profile", "", "write cpu profile to `file`")
	rootCmd.PersistentFlags().StringVar(&traceFlag, "trace", "trace.out", "write execution trace to `file`")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "enable debug mode (prints additional information)")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "V", false, "enable verbose output (prints more detailed information)")
	rootCmd.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "suppress all output except errors")
	rootCmd.Flags().BoolVarP(&versionEnableFlag, "version", "v", false, "show version information")
}
