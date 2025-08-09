package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/style"
	toolsPkg "github.com/yeisme/gocli/pkg/tools"
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
		Example: strings.TrimSpace(`
  gocli tools list
  gocli tools list --json
  gocli tools list --verbose
`),
		Run: func(cmd *cobra.Command, _ []string) {
			listJSON, _ := cmd.Flags().GetBool("json")
			// 优先使用全局 verbose；若未设置，则读取本地 flags
			v := verbose

			gocliToolsPath := gocliCtx.Config.Tools.GoCLIToolsPath
			tools := toolsPkg.FindTools(v, gocliToolsPath)
			if listJSON {
				b, err := json.MarshalIndent(tools, "", "  ")
				if err != nil {
					cmd.PrintErrf("failed to marshal json: %v\n", err)
					return
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
				if err != nil {
					log.Error().Err(err).Msg("failed to print tools list in JSON format")
				}
				return
			}

			// 表格输出
			headers := []string{"name", "source", "path"}
			rows := make([][]string, 0, len(tools))
			for _, t := range tools {
				rows = append(rows, []string{t.Name, string(t.Source), t.Path})
			}
			if err := style.PrintTable(cmd.OutOrStdout(), headers, rows, 0); err != nil {
				cmd.PrintErrf("failed to print table: %v\n", err)
			}
		},
	}
	toolInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install a tool",
		Example: strings.TrimSpace(`
  gocli tools install <tool-name>
  gocli tools install --version <version> <tool-name>
  gocli tools install --global <tool-name>
  gocli tools install --path <path> <tool-name>
`),
	}
	toolUpdateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update a tool",
	}
	toolAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a tool",
	}
	toolUninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall a tool",
	}
	toolSearchCmd = &cobra.Command{
		Use:   "search",
		Short: "Search for a tool",
	}
)

func init() {
	rootCmd.AddCommand(toolsCmd)

	// Add subcommands for tools
	toolsCmd.AddCommand(
		toolListCmd,
		toolInstallCmd,
		toolUpdateCmd,
		toolAddCmd,
		toolUninstallCmd,
		toolSearchCmd,
	)
}
