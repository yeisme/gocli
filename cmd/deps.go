package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

var (
	depsList bool
	depsCmd  = &cobra.Command{
		Use:   "deps [name]",
		Short: "List dependencies",
		Long:  "List all dependencies of the current project, including their versions and any additional information.",
		Run: func(cmd *cobra.Command, args []string) {
			ProcessorWithDefault(TypeDeps, args, depsList,
				func(config *types.Config) []types.Command {
					return config.Deps
				},
				func() {
					v := utils.IsVerbose()
					if v {
						utils.Info("No dependencies found in the project configuration, using default dependency management.")
					}
					utils.Info("Running default dependency management: go mod tidy")
					if err := utils.GoExec("go mod tidy"); err != nil {
						utils.Error(fmt.Sprintf("Failed to execute 'go mod tidy': %v", err))
					} else if v {
						utils.Info("Successfully executed: go mod tidy")
					}
				})
		},
	}
)

func init() {
	depsCmd.Flags().BoolVarP(&depsList, "list", "l", false, "List available deps configurations")
	rootCmd.AddCommand(depsCmd)
}
