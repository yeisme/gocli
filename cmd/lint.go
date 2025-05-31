package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	lintCmd = &cobra.Command{
		Use:   "lint [name]",
		Short: "Run linters",
		Aliases: []string{"l"},
		Long:  "Run code linters and static analysis tools on the project.",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsVerbose() {
				utils.Info("Starting lint process...")
			}

			config := parse.GetProjectConfig()
			if config == nil {
				utils.Error("No project configuration found. Please run 'gocli config init' to create a configuration file.")
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Loaded project configuration: %s v%s", config.Project.Name, config.Project.Version))
				utils.Info(fmt.Sprintf("Found %d lint configurations", len(config.Lint)))
			}

			targetName := "default"
			if len(args) > 0 {
				targetName = args[0]
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Target lint configuration specified: %s", targetName))
				}
			} else if utils.IsVerbose() {
				utils.Info("Using default lint configuration")
			}

			var targetLint *types.Command
			for _, lint := range config.Lint {
				if lint.Name == targetName {
					targetLint = &lint
					break
				}
			}

			if targetLint == nil {
				utils.Error(fmt.Sprintf("Lint configuration '%s' not found", targetName))
				if utils.IsVerbose() {
					utils.Info("Available lint configurations:")
					for _, lint := range config.Lint {
						utils.Info(fmt.Sprintf("  - %s: %s", lint.Name, lint.Description))
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Found lint configuration: %s", targetLint.Name))
				utils.Info(fmt.Sprintf("Description: %s", targetLint.Description))
				utils.Info(fmt.Sprintf("Commands to execute: %d", len(targetLint.Cmd)))
			}

			utils.Info(fmt.Sprintf("Executing lint: %s - %s", targetLint.Name, targetLint.Description))

			for i, cmdStr := range targetLint.Cmd {
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Executing lint command %d/%d: %s", i+1, len(targetLint.Cmd), cmdStr))
				}

				if err := utils.GoExec(cmdStr); err != nil {
					utils.Error(fmt.Sprintf("Lint failed at command '%s': %v", cmdStr, err))
					return
				}

				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Successfully executed lint command %d/%d", i+1, len(targetLint.Cmd)))
				}
			}

			if utils.IsVerbose() {
				utils.Info("Lint process completed successfully")
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(lintCmd)
}
