package cmd

import (
	"fmt"
	"os"
	"runtime/pprof"
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
	globalFlags   = context.GlobalFlags{}
	configPath    = globalFlags.ConfigPath
	debug         = globalFlags.Debug
	verbose       = globalFlags.Verbose
	quiet         = globalFlags.Quiet
	cpuProfile    = globalFlags.CPUProfile
	versionEnable = globalFlags.VersionEnable
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gocli",
	Short: "gocli is a CLI application for managing your Go projects",
	Long:  `gocli is a command line interface application that helps you manage your Go projects efficiently.`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionEnable {
			fmt.Fprintln(cmd.OutOrStdout(), version.GetShortVersionString())
			os.Exit(0)
		}
		if len(args) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No arguments provided")
			_ = cmd.Help()
		}
	},
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		if cpuProfile != "" {
			f, err := os.Create(cpuProfile)
			if err != nil {
				log.Fatal().Err(err).Msg("could not create CPU profile")
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				log.Fatal().Err(err).Msg("could not start CPU profile")
			}
		}
		ctx := context.InitGocliContext(configPath, debug, verbose, quiet)

		gocliCtx = ctx
		log = ctx.Logger

		log.Info().Msgf("Execute Command: %s %s", "gocli", strings.Join(os.Args[1:], " "))
	},
	PersistentPostRun: func(_ *cobra.Command, _ []string) {
		if cpuProfile != "" {
			pprof.StopCPUProfile()
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
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "config file")
	rootCmd.PersistentFlags().StringVar(&cpuProfile, "cpu-profile", "", "write cpu profile to `file`")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode (prints additional information)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "enable verbose output (prints more detailed information)")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress all output except errors")
	rootCmd.Flags().BoolVarP(&versionEnable, "version", "v", false, "show version information")
}
