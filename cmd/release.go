package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	releaseList bool
	releaseCmd  = &cobra.Command{
		Use:   "release [name]",
		Short: "Release the project",
		Long:  "Create a release build and publish the project using the specified release configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			config := parse.GetProjectConfig()
			if config == nil {
				utils.Error("No project configuration found. Please run 'gocli config init' to create a configuration file.")
				return
			}

			// Handle --list flag
			if releaseList {
				utils.Header("Available Release Configurations")
				if len(config.Release) == 0 {
					utils.Info("No release configurations found.")
					return
				}

				for _, release := range config.Release {
					if utils.IsVerbose() {
						utils.Box(fmt.Sprintf("%s - %s",
							release.Name, release.Description),
							fmt.Sprintf("Commands:\n%s", joinStringSlice(release.Cmds)),
							len(release.Name)+len(release.Description)+10)
					} else {
						utils.ListItem("%s - %s", release.Name, release.Description)
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info("Starting release process...")
			}

			targetName := "default"
			if len(args) > 0 {
				targetName = args[0]
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Target release configuration specified: %s", targetName))
				}
			} else if utils.IsVerbose() {
				utils.Info("Using default release configuration")
			}

			var targetRelease *types.Command
			for _, release := range config.Release {
				if release.Name == targetName {
					targetRelease = &release
					break
				}
			}

			if targetRelease == nil {
				utils.Error(fmt.Sprintf("Release configuration '%s' not found", targetName))
				if utils.IsVerbose() {
					utils.Info("Available release configurations:")
					for _, release := range config.Release {
						utils.Info(fmt.Sprintf("  - %s: %s", release.Name, release.Description))
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Found release configuration: %s", targetRelease.Name))
				utils.Info(fmt.Sprintf("Description: %s", targetRelease.Description))
				utils.Info(fmt.Sprintf("Commands to execute: %d", len(targetRelease.Cmds)))
			}

			utils.Info(fmt.Sprintf("Executing release: %s - %s", targetRelease.Name, targetRelease.Description))

			for i, cmdStr := range targetRelease.Cmds {
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Executing release command %d/%d: %s", i+1, len(targetRelease.Cmds), cmdStr))
				}

				if err := utils.GoExec(cmdStr); err != nil {
					utils.Error(fmt.Sprintf("Release failed at command '%s': %v", cmdStr, err))
					return
				}

				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Successfully executed release command %d/%d", i+1, len(targetRelease.Cmds)))
				}
			}

			if utils.IsVerbose() {
				utils.Info("Release process completed successfully")
			}
		},
	}
)

func init() {
	releaseCmd.Flags().BoolVarP(&releaseList, "list", "l", false, "List available release configurations")
	rootCmd.AddCommand(releaseCmd)
}
