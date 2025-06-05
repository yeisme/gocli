package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/types"
)

var (
	cleanList bool
	cleanCmd  = &cobra.Command{
		Use:   "clean [name]",
		Short: "Clean the project",
		Long:  "Clean build artifacts, temporary files, and caches from the project.",
		Run: func(cmd *cobra.Command, args []string) {
			CommandProcessor(TypeClean, args, cleanList, func(config *types.Config) []types.Command {
				return config.Clean
			})
		},
	}
)

func init() {
	cleanCmd.Flags().BoolVarP(&cleanList, "list", "l", false, "List available clean configurations")
	rootCmd.AddCommand(cleanCmd)
}
