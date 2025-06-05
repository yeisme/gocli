package cmd

import (
	"fmt"
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
		Use:     "check",
		Short:   "Check tool availability",
		Long:    "Check if required development tools are installed and available.",
		Aliases: []string{"c"},
		Run: func(cmd *cobra.Command, args []string) {
			v := utils.IsVerbose()
			// Use global projectConfig from root
			config := projectConfig
			if config == nil {
				utils.Error("No configuration found. Please run this command in a project directory or use --user flag.")
				os.Exit(1)
			}

			if v {
				utils.Debug("Starting tool configuration check...")
			}
			if !utils.IsQuiet() {
				utils.Info("Checking tool configuration...")
			}

			// Handle --all flag
			if _all {
				_dev, _go, _git, _custom = true, true, true, true
				if v {
					utils.Debug("All tools flag enabled, checking all tool categories")
				}
			}

			if v {
				utils.Debug("Tool check flags - dev: %v, go: %v, git: %v, custom: %v", _dev, _go, _git, _custom)
			}

			err := tools.ToolsCheck(_dev, _go, _git, _custom, config.Tools)
			if err != nil {
				utils.Error("Tool check failed: %v", err)
				os.Exit(1)
			}

			if !utils.IsQuiet() {
				utils.Success("All tool checks completed")
			}
		},
	}

	toolInstallCmd = &cobra.Command{
		Use:     "install",
		Short:   "Install tools",
		Long:    "Install tools defined in the configuration. Supports Go tools (via 'go install') and Git tools (via git clone + build).",
		Aliases: []string{"i"},
		Run: func(cmd *cobra.Command, args []string) {
			v := utils.IsVerbose()
			// Use global projectConfig from root
			config := projectConfig
			if config == nil {
				utils.Error("No configuration found. Please run this command in a project directory or use --user flag.")
				os.Exit(1)
			}

			if v {
				utils.Debug("Tool installation flags - dev: %v, go: %v, git: %v, custom: %v", _dev, _go, _git, _custom)
			}

			// Handle --all flag
			if _all {
				_dev, _go, _git, _custom = true, true, true, true
				if v {
					utils.Debug("All tools flag enabled, installing all tool categories")
				}
			}

			hasAnyInstallation := false

			// Install Go tools
			if _go && len(config.Tools.Go) > 0 {
				if v {
					utils.Debug("Found %d Go tools configured for installation", len(config.Tools.Go))
				}
				if !utils.IsQuiet() {
					utils.Info("Installing Go tools...")
				}

				err := tools.InstallGoTools(config.Tools.Go)
				if err != nil {
					utils.Error("Go tools installation failed: %v", err)
					os.Exit(1)
				}

				if !utils.IsQuiet() {
					utils.Success("Go tools installation completed")
				}
				hasAnyInstallation = true
			}

			// Install Git tools
			if _git && len(config.Tools.Git) > 0 {
				if v {
					utils.Debug("Found %d Git tools configured for installation", len(config.Tools.Git))
				}
				if !utils.IsQuiet() {
					utils.Info("Installing Git tools...")
				}

				err := tools.InstallGitTools(config.Tools.Git)
				if err != nil {
					utils.Error("Git tools installation failed: %v", err)
					os.Exit(1)
				}
				if !utils.IsQuiet() {
					utils.Success("Git tools installation completed")
				}
				hasAnyInstallation = true
			}

			// Dev and Custom tools are not installable automatically
			if _dev && len(config.Tools.Dev) > 0 {
				if !utils.IsQuiet() {
					utils.Warning("Development tools need to be installed manually")
					utils.Info("Run 'gocli tool check --dev' to see required development tools")
				}
			}
			if _custom && len(config.Tools.Custom) > 0 {
				if !utils.IsQuiet() {
					utils.Warning("Custom tools need to be installed manually")
					utils.Info("Run 'gocli tool list --custom' to see custom tool commands")
				}
			}

			if !hasAnyInstallation {
				if !_dev && !_go && !_git && !_custom && !_all {
					// Default to Go tools if no flags specified
					if len(config.Tools.Go) > 0 {
						if !utils.IsQuiet() {
							utils.Info("No tool type specified, installing Go tools by default...")
						}
						err := tools.InstallGoTools(config.Tools.Go)
						if err != nil {
							utils.Error("Go tools installation failed: %v", err)
							os.Exit(1)
						}
						if !utils.IsQuiet() {
							utils.Success("Go tools installation completed")
						}
					} else {
						if !utils.IsQuiet() {
							utils.Info("No Go tools configured for installation.")
						}
					}
				} else {
					if !utils.IsQuiet() {
						utils.Info("No tools were installed. Use --go or --git flags to specify tool types.")
					}
				}
			}
		},
	}

	toolListCmd = &cobra.Command{
		Use:   "list",
		Short: "List configured tools",
		Long:  "List all tools defined in the configuration file.",
		Run: func(cmd *cobra.Command, args []string) {
			v := utils.IsVerbose()
			// Use global projectConfig from root
			config := projectConfig
			if config == nil {
				utils.Error("No configuration found. Please run this command in a project directory or use --user flag.")
				os.Exit(1)
			}

			if v {
				utils.Debug("Listing configured tools...")
			}
			if !utils.IsQuiet() {
				utils.Header("Configured Tools")
			}

			// Handle --all flag
			if _all {
				_dev, _go, _git, _custom = true, true, true, true
				if v {
					utils.Debug("All tools flag enabled, listing all tool categories")
				}
			}

			hasAnyTools := false

			if _dev && len(config.Tools.Dev) > 0 {
				if v {
					utils.Debug("Found %d development tools", len(config.Tools.Dev))
				}
				if !utils.IsQuiet() {
					utils.SubHeader("Development Tools")
					for _, tool := range config.Tools.Dev {
						version := tool.Version
						if version == "" {
							version = "any"
						}
						utils.ListItem("%s (version: %s)", tool.Name, version)
					}
					utils.Print(utils.White, "\n")
				}
				hasAnyTools = true
			}

			if _go && len(config.Tools.Go) > 0 {
				if v {
					utils.Debug("Found %d Go tools", len(config.Tools.Go))
				}
				if !utils.IsQuiet() {
					utils.SubHeader("Go Tools")
					for _, tool := range config.Tools.Go {
						flags := ""
						if len(tool.Flags) > 0 {
							flags = " (flags: " + joinStringSlice(tool.Flags) + ")"
						}
						utils.ListItem("%s - %s%s", tool.Name, tool.URL, flags)
					}
					utils.Print(utils.White, "\n")
				}
				hasAnyTools = true
			}

			if _git && len(config.Tools.Git) > 0 {
				if v {
					utils.Debug("Found %d Git tools", len(config.Tools.Git))
				}
				if !utils.IsQuiet() {
					utils.SubHeader("Git Tools")
					for _, tool := range config.Tools.Git {
						content := fmt.Sprintf("Repository: %s\nBuild type: %s", tool.URL, tool.Type)
						if tool.Recipe != "" {
							content += fmt.Sprintf("\nRecipe: %s", tool.Recipe)
						}
						utils.Box(fmt.Sprintf("%s - %s", tool.Name, tool.Description), content, len(tool.Description)+len(tool.Name)+20)
						utils.Print(utils.White, "\n")
					}
					utils.Print(utils.White, "\n")
				}
				hasAnyTools = true
			}

			if _custom && len(config.Tools.Custom) > 0 {
				if v {
					utils.Debug("Found %d custom tools", len(config.Tools.Custom))
				}
				if !utils.IsQuiet() {
					utils.SubHeader("Custom Tools")
					for _, tool := range config.Tools.Custom {
						utils.ListItem("%s: %s", tool.Name, tool.Cmds)
						if len(tool.Needs) > 0 {
							utils.ListItem("  Dependencies: %s", joinStringSlice(tool.Needs))
						}
					}
				}
				hasAnyTools = true
			}

			if !hasAnyTools {
				if !utils.IsQuiet() {
					utils.Info("No tools configured or selected.")
				}
				if v {
					utils.Debug("Tool selection flags - dev: %v, go: %v, git: %v, custom: %v", _dev, _go, _git, _custom)
				}
			}
		},
	}
)

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
