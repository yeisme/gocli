package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/types"
)

var (
	testList bool
	testCmd  = &cobra.Command{
		Use:   "test [name]",
		Short: "Run tests",
		Long:  "Run unit tests, integration tests, and other test suites for the project.",
		Run: func(cmd *cobra.Command, args []string) {
			CommandProcessor(TypeTest, args, testList, func(config *types.Config) []types.Command {
				return config.Test
			})
		},
	}
)

func init() {
	testCmd.Flags().BoolVarP(&testList, "list", "l", false, "List available test configurations")
	rootCmd.AddCommand(testCmd)
}
