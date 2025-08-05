package cmd

import "github.com/spf13/cobra"

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage Go projects",
	Long:  `gocli project allows you to manage your Go projects, including creating, building, and running them.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	Aliases: []string{"p", "mod"},
}

func init() {
	rootCmd.AddCommand(projectCmd)

}
