package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/context"
	log2 "github.com/yeisme/gocli/pkg/utils/log"
)

var (
	gocliCtx *context.GocliContext
	log      log2.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gocli",
	Short: "gocli is a CLI application for managing your Go projects",
	Long:  `gocli is a command line interface application that helps you manage your Go projects efficiently.`,
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing config flag: %v\n", err)
			os.Exit(1)
		}
		debug, _ := cmd.Flags().GetBool("debug")
		verbose, _ := cmd.Flags().GetBool("verbose")
		quiet, _ := cmd.Flags().GetBool("quiet")
		version, _ := cmd.Flags().GetBool("version")

		ctx := context.InitGocliContext(configPath)
		ctx.Config.App.Debug = debug
		ctx.Config.App.Verbose = verbose
		ctx.Config.App.Quiet = quiet

		gocliCtx = ctx
		log = ctx.Logger

		log.Info().Msgf("Execute Command: %s %s", "gocli", strings.Join(os.Args[1:], " "))
		if version {
			fmt.Printf("gocli version: v%s\n", gocliCtx.Config.Version)
			os.Exit(0)
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
	rootCmd.PersistentFlags().String("config", "", "config file")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose output")
	rootCmd.PersistentFlags().Bool("quiet", false, "suppress all output except errors")
	rootCmd.PersistentFlags().Bool("version", false, "display the version of the application")
}
