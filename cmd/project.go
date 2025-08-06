package cmd

import "github.com/spf13/cobra"

var (
	projectCmd = &cobra.Command{
		Use:     "project",
		Short:   "Manage Go projects",
		Long:    `gocli project allows you to manage your Go projects, including creating, building, and running them.`,
		Aliases: []string{"p", "mod"},
	}

	projectInitCmd   = &cobra.Command{Use: "init", Short: "Initialize a new Go project"}
	projectBuildCmd  = &cobra.Command{Use: "build", Short: "Build the Go project"}
	projectRunCmd    = &cobra.Command{Use: "run", Short: "Run the Go project"}
	projectListCmd   = &cobra.Command{Use: "list", Short: "List Go projects"}
	projectInfoCmd   = &cobra.Command{Use: "info", Short: "Show information about the Go project"}
	projectAddCmd    = &cobra.Command{Use: "add", Short: "Add a dependency to the Go project"}
	projectTestCmd   = &cobra.Command{Use: "test", Short: "Run tests for the Go project"}
	projectLintCmd   = &cobra.Command{Use: "lint", Short: "Lint the Go project"}
	projectUpdateCmd = &cobra.Command{Use: "update", Short: "Update dependencies of the Go project"}
	projectDepsCmd   = &cobra.Command{Use: "deps", Short: "Manage dependencies of the Go project"}
	projectConfigCmd = &cobra.Command{Use: "config", Short: "Manage project configuration"}
	projectDocCmd    = &cobra.Command{Use: "doc", Short: "Generate documentation for the Go project"}
)

func init() {
	rootCmd.AddCommand(projectCmd)

	projectCmd.AddCommand(
		projectInitCmd,
		projectBuildCmd,
		projectRunCmd,
		projectListCmd,
		projectInfoCmd,
		projectAddCmd,
		projectTestCmd,
		projectLintCmd,
		projectUpdateCmd,
		projectDepsCmd,
		projectConfigCmd,
		projectDocCmd,
	)
}
