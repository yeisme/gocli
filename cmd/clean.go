package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	cleanList bool
	cleanCmd  = &cobra.Command{
		Use:   "clean [name]",
		Short: "Clean the project",
		Long:  "Clean build artifacts, cache, and other temporary files from the project.",
		Run: func(cmd *cobra.Command, args []string) {
			config := parse.GetProjectConfig()
			if config == nil {
				utils.Error("No project configuration found. Please run 'gocli config init' to create a configuration file.")
				return
			}

			// Handle --list flag
			if cleanList {
				utils.Header("Available Clean Configurations")
				if len(config.Clean) == 0 {
					utils.Info("No clean configurations found.")
					return
				}

				for _, clean := range config.Clean {
					if utils.IsVerbose() {
						utils.Box(fmt.Sprintf("%s - %s",
							clean.Name, clean.Description),
							fmt.Sprintf("Commands:\n%s", joinStringSlice(clean.Cmds)),
							len(clean.Name)+len(clean.Description)+10)
					} else {
						utils.ListItem("%s - %s", clean.Name, clean.Description)
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info("Starting clean process...")
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Loaded project configuration: %s v%s", config.Project.Name, config.Project.Version))
				utils.Info(fmt.Sprintf("Found %d clean configurations", len(config.Clean)))
			}

			targetName := "default"
			if len(args) > 0 {
				targetName = args[0]
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Target clean configuration specified: %s", targetName))
				}
			} else if utils.IsVerbose() {
				utils.Info("Using default clean configuration")
			}

			var targetClean *types.Command
			for _, clean := range config.Clean {
				if clean.Name == targetName {
					targetClean = &clean
					break
				}
			}

			if targetClean == nil {
				utils.Error(fmt.Sprintf("Clean configuration '%s' not found", targetName))
				if utils.IsVerbose() {
					utils.Info("Available clean configurations:")
					for _, clean := range config.Clean {
						utils.Info(fmt.Sprintf("  - %s: %s", clean.Name, clean.Description))
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Found clean configuration: %s", targetClean.Name))
				utils.Info(fmt.Sprintf("Description: %s", targetClean.Description))
				utils.Info(fmt.Sprintf("Commands to execute: %d", len(targetClean.Cmds)))
			}

			utils.Info(fmt.Sprintf("Executing clean: %s - %s", targetClean.Name, targetClean.Description))

			for i, cmdStr := range targetClean.Cmds {
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Executing clean command %d/%d: %s", i+1, len(targetClean.Cmds), cmdStr))
				}

				if err := utils.GoExec(cmdStr); err != nil {
					utils.Error(fmt.Sprintf("Clean failed at command '%s': %v", cmdStr, err))
					return
				}

				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Successfully executed clean command %d/%d", i+1, len(targetClean.Cmds)))
				}
			}

			if utils.IsVerbose() {
				utils.Info("Clean process completed successfully")
			}
		},
	}
)

func init() {
	cleanCmd.Flags().BoolVarP(&cleanList, "list", "l", false, "List available clean configurations")
	rootCmd.AddCommand(cleanCmd)
}
