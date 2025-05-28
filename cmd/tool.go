package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// toolCmd represents the tool command
var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("tool called")
	},
}

func init() {
	rootCmd.AddCommand(toolCmd)
}
