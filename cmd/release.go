package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/types"
)

var (
	releaseList bool
	releaseCmd  = &cobra.Command{
		Use:   "release [name]",
		Short: "Release the project",
		Long:  "Create a release build and publish the project using the specified release configuration.",
		Run: func(cmd *cobra.Command, args []string) {
			CommandProcessor(TypeRelease, args, releaseList, func(config *types.Config) []types.Command {
				return config.Release
			})
		},
	}
)

func init() {
	releaseCmd.Flags().BoolVarP(&releaseList, "list", "l", false, "List available release configurations")
	rootCmd.AddCommand(releaseCmd)
}
