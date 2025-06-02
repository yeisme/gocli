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

	// Global flags
	verbose       bool
	color         bool
	quiet         bool
	version       string = "0.1.0"
	user          bool
	configfile    string
	userConfig    *types.Config
	projectConfig *types.Config

	// Global variables
	targetName = "default"

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

			// Load user configuration
			userConfig = parse.UserConfig()

			// Load and set project configuration
			if !user {
				configPath := utils.ProjectConfigPath(utils.GetConfigFile())
				if configPath != "" {
					if config, err := parse.ParseConfigFromFile(configPath); err == nil {
						parse.SetProjectConfig(config)
						projectConfig = config
					}
				}
			} else {
				// When --user flag is set, use userConfig as projectConfig
				projectConfig = userConfig
			}

			// Fallback: if no project config and not using --user flag, try to load from parse
			if projectConfig == nil && !user {
				projectConfig = parse.ProjectConfig()
			}
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

// joinStringSlice joins string slice with commas
func joinStringSlice(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += ", " + slice[i]
	}
	return result
}
