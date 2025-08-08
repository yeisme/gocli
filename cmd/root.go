package cmd

import (
	"os"
	"runtime/pprof"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/context"
	log2 "github.com/yeisme/gocli/pkg/utils/log"
)

var (
	gocliCtx *context.GocliContext
	log      log2.Logger

	// Global flags
	configPath string
	debug      bool
	verbose    bool
	quiet      bool
	cpuProfile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gocli",
	Short: "gocli is a CLI application for managing your Go projects",
	Long:  `gocli is a command line interface application that helps you manage your Go projects efficiently.`,
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
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose output (prints more detailed information)")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress all output except errors")
}
