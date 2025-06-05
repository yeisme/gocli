package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/types"
)

var (
	devList bool
	devCmd = &cobra.Command{
		Use:   "dev [name]",
		Short: "Run development mode",
		Aliases: []string{"d", "hot"},
		Long:  "Run the project in development mode with hot reload and other development features.",
		Run: func(cmd *cobra.Command, args []string) {
			CommandProcessor(TypeDev, args, devList, func(config *types.Config) []types.Command {
				return config.Dev
			})
		},
	}
)

func init() {
	devCmd.Flags().BoolVarP(&devList, "list", "l", false, "List available dev configurations")
	rootCmd.AddCommand(devCmd)
}
