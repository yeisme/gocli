package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/parse"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	testList bool
	testCmd  = &cobra.Command{
		Use:   "test [name]",
		Short: "Run tests",
		Long:  "Run unit tests, integration tests, and other test suites for the project.",
		Run: func(cmd *cobra.Command, args []string) {
			config := parse.GetProjectConfig()
			if config == nil {
				utils.Error("No project configuration found. Please run 'gocli config init' to create a configuration file.")
				return
			}

			// Handle --list flag
			if testList {
				utils.Header("Available Test Configurations")
				if len(config.Test) == 0 {
					utils.Info("No test configurations found.")
					return
				}

				for _, test := range config.Test {
					if utils.IsVerbose() {
						utils.Box(fmt.Sprintf("%s - %s",
							test.Name, test.Description),
							fmt.Sprintf("Commands:\n%s", joinStringSlice(test.Cmds)),
							len(test.Name)+len(test.Description)+10)
					} else {
						utils.ListItem("%s - %s", test.Name, test.Description)
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info("Starting test process...")
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Loaded project configuration: %s v%s", config.Project.Name, config.Project.Version))
				utils.Info(fmt.Sprintf("Found %d test configurations", len(config.Test)))
			}

			targetName := "default"
			if len(args) > 0 {
				targetName = args[0]
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Target test configuration specified: %s", targetName))
				}
			} else if utils.IsVerbose() {
				utils.Info("Using default test configuration")
			}

			var targetTest *types.Command
			for _, test := range config.Test {
				if test.Name == targetName {
					targetTest = &test
					break
				}
			}

			if targetTest == nil {
				utils.Error(fmt.Sprintf("Test configuration '%s' not found", targetName))
				if utils.IsVerbose() {
					utils.Info("Available test configurations:")
					for _, test := range config.Test {
						utils.Info(fmt.Sprintf("  - %s: %s", test.Name, test.Description))
					}
				}
				return
			}

			if utils.IsVerbose() {
				utils.Info(fmt.Sprintf("Found test configuration: %s", targetTest.Name))
				utils.Info(fmt.Sprintf("Description: %s", targetTest.Description))
				utils.Info(fmt.Sprintf("Commands to execute: %d", len(targetTest.Cmds)))
			}

			utils.Info(fmt.Sprintf("Executing test: %s - %s", targetTest.Name, targetTest.Description))

			for i, cmdStr := range targetTest.Cmds {
				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Executing test command %d/%d: %s", i+1, len(targetTest.Cmds), cmdStr))
				}

				if err := utils.GoExec(cmdStr); err != nil {
					utils.Error(fmt.Sprintf("Test failed at command '%s': %v", cmdStr, err))
					return
				}

				if utils.IsVerbose() {
					utils.Info(fmt.Sprintf("Successfully executed test command %d/%d", i+1, len(targetTest.Cmds)))
				}
			}

			if utils.IsVerbose() {
				utils.Info("Test process completed successfully")
			}
		},
	}
)

func init() {
	testCmd.Flags().BoolVarP(&testList, "list", "l", false, "List available test configurations")
	rootCmd.AddCommand(testCmd)
}
