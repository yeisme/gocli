package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/context"
)

var (
	gocliCtx *context.GocliContext
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gocli",
	Short: "gocli is a CLI application for managing your Go projects",
	Long:  `gocli is a command line interface application that helps you manage your Go projects efficiently.`,
	Run: func(cmd *cobra.Command, args []string) {

	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		debug, _ := cmd.Flags().GetBool("debug")
		verbose, _ := cmd.Flags().GetBool("verbose")
		quiet, _ := cmd.Flags().GetBool("quiet")

		ctx := context.InitGocliContext(configPath)
		ctx.Config.App.Debug = debug
		ctx.Config.App.Verbose = verbose
		ctx.Config.App.Quiet = quiet

		gocliCtx = ctx

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	version, _ := rootCmd.Flags().GetBool("version")
	if version {
		rootCmd.Version = gocliCtx.Config.Version
		fmt.Printf("gocli version: v%s\n", rootCmd.Version)
		os.Exit(0)
	}

	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.gocli.yaml)")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose output")
	rootCmd.PersistentFlags().Bool("quiet", false, "suppress all output except errors")
	rootCmd.PersistentFlags().Bool("version", false, "display the version of the application")
}
