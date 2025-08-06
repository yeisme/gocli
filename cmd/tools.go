package cmd

import (
	"github.com/spf13/cobra"
)

var (
	toolsCmd = &cobra.Command{
		Use:     "tools",
		Short:   "Tools Management for gocli",
		Long:    `gocli tools allows you to manage various tools and utilities that enhance your golang development experience.`,
		Aliases: []string{"tool", "t"},
	}

	toolListCmd = &cobra.Command{
		Use:   "list",
		Short: "List available tools",
		Long:  `gocli tools list displays all available tools that can be used with gocli.`,
	}
)

func init() {
	rootCmd.AddCommand(toolsCmd)

	// Add subcommands for tools
	toolsCmd.AddCommand(
		toolListCmd,
		// toolInstallCmd,
		// toolUpdateCmd,
		// toolAddCmd,
		// toolRemoveCmd,
	)
}
