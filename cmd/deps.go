package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	depsCmd = &cobra.Command{
		Use:   "deps [name]",
		Short: "List dependencies",
		Long:  "List all dependencies of the current project, including their versions and any additional information.",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsVerbose() {
				utils.Info("Starting dependency management...")
			}

			config := parse.GetProjectConfig()
			if config == nil {
				utils.Error("No project configuration found. Please run 'gocli config init' to create a configuration file.")
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Loaded project configuration: %s v%s", config.Project.Name, config.Project.Version))
				utils.Info(fmt.Sprintf("Found %d dependency configurations", len(config.Deps)))
			}

			if len(config.Deps) == 0 {
				if utils.IsVerbose() {
					utils.Info("No dependencies found in the project configuration, using default dependency management.")
				}
				utils.Info("Running default dependency management: go mod tidy")
				if err := utils.GoExec("go mod tidy"); err != nil {
					utils.Error(fmt.Sprintf("Failed to execute 'go mod tidy': %v", err))
				} else if utils.IsVerbose() {
					utils.Info("Successfully executed: go mod tidy")
				}
				return
			}

			targetName := "default"
			if len(args) > 0 {
				targetName = args[0]
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Target dependency configuration specified: %s", targetName))
				}
			} else if utils.IsVerbose() {
				utils.Info("Using default dependency configuration")
			}

			var targetDeps *types.Command
			for _, dep := range config.Deps {
				if dep.Name == targetName {
					targetDeps = &dep
					break
				}
			}

			if targetDeps == nil {
				utils.Error(fmt.Sprintf("Deps configuration '%s' not found", targetName))
				if utils.IsVerbose() {
					utils.Info("Available dependency configurations:")
					for _, dep := range config.Deps {
						utils.Info(fmt.Sprintf("  - %s: %s", dep.Name, dep.Description))
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Found dependency configuration: %s", targetDeps.Name))
				utils.Info(fmt.Sprintf("Description: %s", targetDeps.Description))
				utils.Info(fmt.Sprintf("Commands to execute: %d", len(targetDeps.Cmds)))
			}

			utils.Info(fmt.Sprintf("Executing deps: %s - %s", targetDeps.Name, targetDeps.Description))

			for i, cmdStr := range targetDeps.Cmds {
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Executing command %d/%d: %s", i+1, len(targetDeps.Cmds), cmdStr))
				}

				if err := utils.GoExec(cmdStr); err != nil {
					utils.Error(fmt.Sprintf("Failed to execute '%s': %v", cmdStr, err))
					return
				}

				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Successfully executed command %d/%d", i+1, len(targetDeps.Cmds)))
				}
			}

			if utils.IsVerbose() {
				utils.Info("Dependency management completed successfully")
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(depsCmd)
}
