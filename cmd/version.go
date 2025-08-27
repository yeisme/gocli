package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/utils/version"
)

var (
	// Version command flags
	versionDetailed bool
	versionJSON     bool
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `
Display version information for gocli.

Examples:
  # Show short version info (default)
  gocli version

  # Show detailed version info
  gocli version --detailed

  # Show version info in JSON format
  gocli version --json

Notes:
  - By default, shows a short version string similar to GitHub CLI.
  - Use --detailed flag to get more comprehensive version information like golangci-lint.
  - Use --json flag to output version information in JSON format.`,
	Run: func(cmd *cobra.Command, _ []string) {
		if versionJSON {
			info := version.GetVersion()
			output, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				cmd.PrintErrf("Error formatting JSON: %v\n", err)
				return
			}
			fmt.Println(string(output))
		} else if versionDetailed {
			fmt.Println(version.GetVersionString())
		} else {
			fmt.Println(version.GetShortVersionString())
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().BoolVarP(&versionDetailed, "detailed", "d", false, "show detailed version information")
	versionCmd.Flags().BoolVarP(&versionJSON, "json", "j", false, "output version information in JSON format")
}
