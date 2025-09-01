package cmd

import "github.com/spf13/cobra"

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a command",
	Long:  `Run a command in the specified environment`,
}

func init() {
	rootCmd.AddCommand(runCmd)
}
