package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	helpCmd = &cobra.Command{
		Use:   "help [name]",
		Short: "Show help information",
		Long:  "Show help information for the project or specific commands.",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsVerbose() {
				utils.Info("Starting help process...")
			}

			config := parse.GetProjectConfig()
			if config == nil {
				utils.Error("No project configuration found. Please run 'gocli config init' to create a configuration file.")
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Loaded project configuration: %s v%s", config.Project.Name, config.Project.Version))
				utils.Info(fmt.Sprintf("Found %d help configurations", len(config.Help)))
			}

			targetName := "default"
			if len(args) > 0 {
				targetName = args[0]
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Target help configuration specified: %s", targetName))
				}
			} else if utils.IsVerbose() {
				utils.Info("Using default help configuration")
			}

			var targetHelp *types.Command
			for _, help := range config.Help {
				if help.Name == targetName {
					targetHelp = &help
					break
				}
			}

			if targetHelp == nil {
				utils.Error(fmt.Sprintf("Help configuration '%s' not found", targetName))
				if utils.IsVerbose() {
					utils.Info("Available help configurations:")
					for _, help := range config.Help {
						utils.Info(fmt.Sprintf("  - %s: %s", help.Name, help.Description))
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Found help configuration: %s", targetHelp.Name))
				utils.Info(fmt.Sprintf("Description: %s", targetHelp.Description))
				utils.Info(fmt.Sprintf("Commands to execute: %d", len(targetHelp.Cmd)))
			}

			utils.Info(fmt.Sprintf("Executing help: %s - %s", targetHelp.Name, targetHelp.Description))

			for i, cmdStr := range targetHelp.Cmd {
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Executing help command %d/%d: %s", i+1, len(targetHelp.Cmd), cmdStr))
				}

				if err := utils.GoExec(cmdStr); err != nil {
					utils.Error(fmt.Sprintf("Help failed at command '%s': %v", cmdStr, err))
					return
				}

				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Successfully executed help command %d/%d", i+1, len(targetHelp.Cmd)))
				}
			}

			if utils.IsVerbose() {
				utils.Info("Help process completed successfully")
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(helpCmd)
}
