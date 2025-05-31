package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/tools"
	"github.com/yeisme/gocli/pkg/utils"
)

// toolCmd represents the tool command
var (
	_dev    bool
	_go     bool
	_git    bool
	_custom bool
	_all    bool

	toolCmd = &cobra.Command{
		Use:   "tool",
		Short: "Manage development tools",
		Long:  "Check, install and manage development tools required for the project.",
		Run: func(cmd *cobra.Command, args []string) {
			// Default behavior: show available subcommands
			cmd.Help()
		},
		Aliases: []string{"t", "tools"},
	}

	toolCheckCmd = &cobra.Command{
		Use:   "check",
		Short: "Check tool availability",
		Long:  "Check if required development tools are installed and available.",
		Run: func(cmd *cobra.Command, args []string) {
			// Use global projectConfig from root
			config := projectConfig
			if config == nil {
				utils.Error("No configuration found. Please run this command in a project directory or use --user flag.")
				os.Exit(1)
			}

			utils.Debug("Starting tool configuration check...")
			utils.Info("Checking tool configuration...")

			// Handle --all flag
			if _all {
				_dev, _go, _git, _custom = true, true, true, true
				utils.Debug("All tools flag enabled, checking all tool categories")
			}

			utils.Debug("Tool check flags - dev: %v, go: %v, git: %v, custom: %v", _dev, _go, _git, _custom)

			err := tools.ToolsCheck(_dev, _go, _git, _custom, config.Tools)
			if err != nil {
				utils.Error("Tool check failed: %v", err)
				os.Exit(1)
			}

			utils.Success("All tool checks completed")
		},
	}

	toolInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install Go tools",
		Long:  "Install Go tools defined in the configuration using 'go install'.",
		Run: func(cmd *cobra.Command, args []string) {
			// Use global projectConfig from root
			config := projectConfig
			if config == nil {
				utils.Error("No configuration found. Please run this command in a project directory or use --user flag.")
				os.Exit(1)
			}

			// Only Go tools can be installed automatically
			if !_go && !_all {
				utils.Warning("Only Go tools can be installed automatically. Use --go or --all flag.")
				return
			}

			if len(config.Tools.Go) == 0 {
				utils.Info("No Go tools configured for installation.")
				return
			}

			utils.Debug("Found %d Go tools configured for installation", len(config.Tools.Go))
			utils.Info("Installing Go tools...")

			err := tools.InstallGoTools(config.Tools.Go)
			if err != nil {
				utils.Error("Tool installation failed: %v", err)
				os.Exit(1)
			}

			utils.Success("Go tools installation completed")
		},
	}

	toolListCmd = &cobra.Command{
		Use:   "list",
		Short: "List configured tools",
		Long:  "List all tools defined in the configuration file.",
		Run: func(cmd *cobra.Command, args []string) {
			// Use global projectConfig from root
			config := projectConfig
			if config == nil {
				utils.Error("No configuration found. Please run this command in a project directory or use --user flag.")
				os.Exit(1)
			}

			utils.Debug("Listing configured tools...")
			utils.Header("Configured Tools")

			// Handle --all flag
			if _all {
				_dev, _go, _git, _custom = true, true, true, true
				utils.Debug("All tools flag enabled, listing all tool categories")
			}

			hasAnyTools := false

			if _dev && len(config.Tools.Dev) > 0 {
				utils.Debug("Found %d development tools", len(config.Tools.Dev))
				utils.SubHeader("Development Tools")
				for _, tool := range config.Tools.Dev {
					version := tool.Version
					if version == "" {
						version = "any"
					}
					utils.ListItem("%s (version: %s)", tool.Name, version)
				}
				hasAnyTools = true
				utils.Print(utils.White, "\n")
			}

			if _go && len(config.Tools.Go) > 0 {
				utils.Debug("Found %d Go tools", len(config.Tools.Go))
				utils.SubHeader("Go Tools")
				for _, tool := range config.Tools.Go {
					flags := ""
					if len(tool.Flags) > 0 {
						flags = " (flags: " + joinStringSlice(tool.Flags) + ")"
					}
					utils.ListItem("%s - %s%s", tool.Name, tool.URL, flags)
				}
				hasAnyTools = true
				utils.Print(utils.White, "\n")
			}

			if _git && len(config.Tools.Git) > 0 {
				utils.Debug("Found %d Git tools", len(config.Tools.Git))
				utils.SubHeader("Git Tools")
				for _, tool := range config.Tools.Git {
					utils.ListItem("%s - %s", tool.Name, tool.Description)
					utils.ListItem("  Repository: %s", tool.URL)
					utils.ListItem("  Build type: %s", tool.Type)
					if tool.Recipe != "" {
						utils.ListItem("  Recipe: %s", tool.Recipe)
					}
				}
				hasAnyTools = true
				utils.Print(utils.White, "\n")
			}

			if _custom && len(config.Tools.Custom) > 0 {
				utils.Debug("Found %d custom tools", len(config.Tools.Custom))
				utils.SubHeader("Custom Tools")
				for _, tool := range config.Tools.Custom {
					utils.ListItem("%s: %s", tool.Name, tool.Cmd)
					if len(tool.Needs) > 0 {
						utils.ListItem("  Dependencies: %s", joinStringSlice(tool.Needs))
					}
				}
				hasAnyTools = true
			}

			if !hasAnyTools {
				utils.Info("No tools configured or selected.")
				utils.Debug("Tool selection flags - dev: %v, go: %v, git: %v, custom: %v", _dev, _go, _git, _custom)
			}
		},
	}
)

// joinStringSlice joins string slice with commas
func joinStringSlice(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += ", " + slice[i]
	}
	return result
}

func init() {
	// Add persistent flags that apply to all subcommands - defaults to false
	toolCmd.PersistentFlags().BoolVar(&_dev, "dev", false, "Include development tools")
	toolCmd.PersistentFlags().BoolVar(&_go, "go", false, "Include Go tools")
	toolCmd.PersistentFlags().BoolVar(&_git, "git", false, "Include Git tools")
	toolCmd.PersistentFlags().BoolVar(&_custom, "custom", false, "Include custom tools")
	toolCmd.PersistentFlags().BoolVarP(&_all, "all", "a", false, "Include all tools (overrides other flags)")

	// 修复 PreRun 函数，正确处理 flag 逻辑
	toolCheckCmd.PreRun = func(cmd *cobra.Command, args []string) {
		// 如果没有指定任何工具类型 flag，则默认检查所有类型
		if !_dev && !_go && !_git && !_custom && !_all {
			_dev, _go, _git, _custom = true, true, true, true
		}
	}

	toolInstallCmd.PreRun = func(cmd *cobra.Command, args []string) {
		// 如果没有指定任何工具类型 flag，则默认只安装 Go 工具
		if !_dev && !_go && !_git && !_custom && !_all {
			_go = true
		}
	}

	toolListCmd.PreRun = func(cmd *cobra.Command, args []string) {
		// 如果没有指定任何工具类型 flag，则默认列出所有类型
		if !_dev && !_go && !_git && !_custom && !_all {
			_dev, _go, _git, _custom = true, true, true, true
		}
	}

	// Add subcommands to tool command
	toolCmd.AddCommand(
		toolCheckCmd,
		toolInstallCmd,
		toolListCmd,
	)

	// Add tool command to root
	rootCmd.AddCommand(toolCmd)
}
