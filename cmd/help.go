package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/types"
)

var (
	helpList bool
	helpCmd  = &cobra.Command{
		Use:   "help [name]",
		Short: "Show help information",
		Long:  "Show help information for the project or specific commands.",
		Run: func(cmd *cobra.Command, args []string) {
			CommandProcessor(TypeHelp, args, helpList, func(config *types.Config) []types.Command {
				return config.Help
			})
		},
	}
)

func init() {
	helpCmd.Flags().BoolVarP(&helpList, "list", "l", false, "List available help configurations")
	rootCmd.AddCommand(helpCmd)
}
