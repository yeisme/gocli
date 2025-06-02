package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	buildList bool
	buildCmd  = &cobra.Command{
		Use:   "build [name]",
		Short: "Build the project",
		Long:  "Build the project using the specified build configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			config := parse.GetProjectConfig()
			if config == nil {
				utils.Error("No project configuration found. Please run 'gocli config init' to create a configuration file.")
				return
			}

			// Handle --list flag
			if buildList {
				utils.Header("Available Build Configurations")
				if len(config.Build) == 0 {
					utils.Info("No build configurations found.")
					return
				}

				for _, build := range config.Build {
					if utils.IsVerbose() {
						utils.Box(fmt.Sprintf("%s - %s",
							build.Name, build.Description),
							fmt.Sprintf("Commands:\n%s", joinStringSlice(build.Cmds)),
							len(build.Name)+len(build.Description)+10)
					} else {
						utils.ListItem("%s - %s", build.Name, build.Description)
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info("Starting build process...")
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Loaded project configuration: %s v%s", config.Project.Name, config.Project.Version))
				utils.Info(fmt.Sprintf("Go version: %s", config.Project.GoVersion))
				utils.Info(fmt.Sprintf("Found %d build configurations", len(config.Build)))
			}

			targetName := "default"
			if len(args) > 0 {
				targetName = args[0]
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Target build configuration specified: %s", targetName))
				}
			} else if utils.IsVerbose() {
				utils.Info("Using default build configuration")
			}

			var targetBuild *types.Command
			for _, build := range config.Build {
				if build.Name == targetName {
					targetBuild = &build
					break
				}
			}

			if targetBuild == nil {
				utils.Error(fmt.Sprintf("Build configuration '%s' not found", targetName))
				if utils.IsVerbose() {
					utils.Info("Available build configurations:")
					for _, build := range config.Build {
						utils.Info(fmt.Sprintf("  - %s: %s", build.Name, build.Description))
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Found build configuration: %s", targetBuild.Name))
				utils.Info(fmt.Sprintf("Description: %s", targetBuild.Description))
				utils.Info(fmt.Sprintf("Commands to execute: %d", len(targetBuild.Cmds)))
			}

			utils.Info(fmt.Sprintf("Executing build: %s - %s", targetBuild.Name, targetBuild.Description))

			for i, cmdStr := range targetBuild.Cmds {
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Executing build command %d/%d: %s", i+1, len(targetBuild.Cmds), cmdStr))
				}

				if err := utils.GoExec(cmdStr); err != nil {
					utils.Error(fmt.Sprintf("Build failed at command '%s': %v", cmdStr, err))
					return
				}

				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Successfully executed build command %d/%d", i+1, len(targetBuild.Cmds)))
				}
			}

			if utils.IsVerbose() {
				utils.Info("Build process completed successfully")
			}
		},
	}
)

func init() {
	buildCmd.Flags().BoolVarP(&buildList, "list", "l", false, "List available build configurations")
	rootCmd.AddCommand(buildCmd)

}
