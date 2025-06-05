package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/types"
)

var (
	runList bool
	runCmd = &cobra.Command{
		Use:   "run [name]",
		Short: "Run the project",
		Long:  "Run the project using the specified run configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			CommandProcessor(TypeRun, args, runList, func(config *types.Config) []types.Command {
				return config.Run
			})
		},
	}
)

func init() {
	runCmd.Flags().BoolVarP(&runList, "list", "l", false, "List available run configurations")
	rootCmd.AddCommand(runCmd)
}
