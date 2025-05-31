package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	devCmd = &cobra.Command{
		Use:   "dev [name]",
		Short: "Run development mode",
		Long:  "Run the project in development mode with hot reload and other development features.",
		Run: func(cmd *cobra.Command, args []string) {
			if utils.IsVerbose() {
				utils.Info("Starting development mode...")
			}

			config := parse.GetProjectConfig()
			if config == nil {
				utils.Error("No project configuration found. Please run 'gocli config init' to create a configuration file.")
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Loaded project configuration: %s v%s", config.Project.Name, config.Project.Version))
				utils.Info(fmt.Sprintf("Found %d dev configurations", len(config.Dev)))
			}

			targetName := "default"
			if len(args) > 0 {
				targetName = args[0]
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Target dev configuration specified: %s", targetName))
				}
			} else if utils.IsVerbose() {
				utils.Info("Using default dev configuration")
			}

			var targetDev *types.Command
			for _, dev := range config.Dev {
				if dev.Name == targetName {
					targetDev = &dev
					break
				}
			}

			if targetDev == nil {
				utils.Error(fmt.Sprintf("Dev configuration '%s' not found", targetName))
				if utils.IsVerbose() {
					utils.Info("Available dev configurations:")
					for _, dev := range config.Dev {
						utils.Info(fmt.Sprintf("  - %s: %s", dev.Name, dev.Description))
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Found dev configuration: %s", targetDev.Name))
				utils.Info(fmt.Sprintf("Description: %s", targetDev.Description))
				utils.Info(fmt.Sprintf("Commands to execute: %d", len(targetDev.Cmd)))
				utils.Info("Development mode may run continuously - use Ctrl+C to stop")
			}

			utils.Info(fmt.Sprintf("Executing dev: %s - %s", targetDev.Name, targetDev.Description))

			for i, cmdStr := range targetDev.Cmd {
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Executing dev command %d/%d: %s", i+1, len(targetDev.Cmd), cmdStr))
				}

				if err := utils.GoExec(cmdStr); err != nil {
					utils.Error(fmt.Sprintf("Dev failed at command '%s': %v", cmdStr, err))
					return
				}

				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Successfully executed dev command %d/%d", i+1, len(targetDev.Cmd)))
				}
			}

			if utils.IsVerbose() {
				utils.Info("Dev process completed")
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(devCmd)
}
