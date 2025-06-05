package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/types"
)

var (
	lintList bool
	lintCmd  = &cobra.Command{
		Use:     "lint [name]",
		Short:   "Run linters",
		Aliases: []string{"l"},
		Long:    "Run code linters and static analysis tools on the project.",
		Run: func(cmd *cobra.Command, args []string) {
			CommandProcessor(TypeLint, args, lintList, func(config *types.Config) []types.Command {
				return config.Lint
			})
		},
	}
)

func init() {
	lintCmd.Flags().BoolVarP(&lintList, "list", "l", false, "List available lint configurations")
	rootCmd.AddCommand(lintCmd)
}
