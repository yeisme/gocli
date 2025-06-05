package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/types"
)

var (
	buildList bool
	buildCmd  = &cobra.Command{
		Use:   "build [name]",
		Short: "Build the project",
		Long:  "Build the project using the specified build configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			CommandProcessor(TypeBuild, args, buildList, func(config *types.Config) []types.Command {
				return config.Build
			})
		},
	}
)

func init() {
	buildCmd.Flags().BoolVarP(&buildList, "list", "l", false, "List available build configurations")
	rootCmd.AddCommand(buildCmd)
}
