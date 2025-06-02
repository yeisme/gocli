package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	runList bool
	runCmd = &cobra.Command{
		Use:   "run [name]",
		Short: "Run the project",
		Long:  "Run the project using the specified run configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			config := parse.GetProjectConfig()
			if config == nil {
				utils.Error("No project configuration found. Please run 'gocli config init' to create a configuration file.")
				return
			}

			// Handle --list flag
			if runList {
				utils.Header("Available Run Configurations")
				if len(config.Run) == 0 {
					utils.Info("No run configurations found.")
					return
				}

				for _, run := range config.Run {
					if utils.IsVerbose() {
						utils.Box(fmt.Sprintf("%s - %s",
							run.Name, run.Description),
							fmt.Sprintf("Commands:\n%s", joinStringSlice(run.Cmds)),
							len(run.Name)+len(run.Description)+10)
					} else {
						utils.ListItem("%s - %s", run.Name, run.Description)
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info("Starting run process...")
				utils.Info(fmt.Sprintf("Loaded project configuration: %s v%s", config.Project.Name, config.Project.Version))
				utils.Info(fmt.Sprintf("Found %d run configurations", len(config.Run)))
			}

			targetName := "default"
			if len(args) > 0 {
				targetName = args[0]
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Target run configuration specified: %s", targetName))
				}
			} else if utils.IsVerbose() {
				utils.Info("Using default run configuration")
			}

			var targetRun *types.Command
			for _, run := range config.Run {
				if run.Name == targetName {
					targetRun = &run
					break
				}
			}

			if targetRun == nil {
				utils.Error(fmt.Sprintf("Run configuration '%s' not found", targetName))
				if utils.IsVerbose() {
					utils.Info("Available run configurations:")
					for _, run := range config.Run {
						utils.Info(fmt.Sprintf("  - %s: %s", run.Name, run.Description))
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Found run configuration: %s", targetRun.Name))
				utils.Info(fmt.Sprintf("Description: %s", targetRun.Description))
				utils.Info(fmt.Sprintf("Commands to execute: %d", len(targetRun.Cmds)))
			}

			utils.Info(fmt.Sprintf("Executing run: %s - %s", targetRun.Name, targetRun.Description))

			for i, cmdStr := range targetRun.Cmds {
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Executing run command %d/%d: %s", i+1, len(targetRun.Cmds), cmdStr))
				}

				if err := utils.GoExec(cmdStr); err != nil {
					utils.Error(fmt.Sprintf("Run failed at command '%s': %v", cmdStr, err))
					return
				}

				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Successfully executed run command %d/%d", i+1, len(targetRun.Cmds)))
				}
			}

			if utils.IsVerbose() {
				utils.Info("Run process completed successfully")
			}
		},
	}
)

func init() {
	runCmd.Flags().BoolVarP(&runList, "list", "l", false, "List available run configurations")
	rootCmd.AddCommand(runCmd)
}
