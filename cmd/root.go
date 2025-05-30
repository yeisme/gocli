package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	verbose    bool
	color      bool
	quiet      bool
	version    string = "0.1.0"
	user       bool
	configfile string
	userConfig *types.Config
	projectConfig *types.Config

	rootCmd = &cobra.Command{
		Use:   "gocli",
		Short: "A powerful CLI tool for Go project management",
		Long: `gocli is a feature-rich command-line interface tool designed to streamline
Go project development and management. It provides configuration management,
build automation, and various utilities to enhance your development workflow.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			utils.SetGlobalFlags(verbose, color, quiet, user)
			utils.SetConfigFile(configfile)
			userConfig = parse.UserConfig()
			projectConfig = parse.ProjectConfig()
		},
		Version: fmt.Sprintf("%s (%s)", version, time.Now().Format("2006-01-02")),
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&color, "color", "c", true, "Enable colored output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all output except errors")
	rootCmd.PersistentFlags().BoolVarP(&user, "user", "u", false, "Set the user config for the command")
	rootCmd.PersistentFlags().StringVarP(&configfile, "config", "f", "", "Path to the configuration file")
}
