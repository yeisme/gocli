package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/models"
	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/plugin"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long:  `gocli plugin allows you to manage plugins for gocli, including listing and installing them.`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available plugins",
	Long: `
List all available gocli plugins from various sources including:
  - User home directory (~/.gocli/plugins)
  - Current directory (./.gocli/plugins)
  - Config file specified path

Examples:
  gocli plugin list
  gocli plugin list --global
  gocli plugin list --source user
`,
	Run: func(cmd *cobra.Command, _ []string) {
		global, _ := cmd.Flags().GetBool("global")
		source, _ := cmd.Flags().GetString("source")

		if !isValidSource(source) {
			log.Error().Msgf("Invalid source: %s. Valid sources are: user, current, config", source)
			return
		}

		listPlugins(global, verbose, source)
	},
}

func isValidSource(source string) bool {
	switch source {
	case "user", "current", "config", "":
		return true
	default:
		return false
	}
}

func listPlugins(global, verbose bool, sourceFilter string) {
	// 获取配置的插件路径
	var configPluginPath string
	if gocliCtx != nil && gocliCtx.Config.Plugin.DirPath != "" && gocliCtx.Config.Plugin.Enabled {
		configPluginPath = gocliCtx.Config.Plugin.DirPath
	}

	// 创建插件管理器
	pm := plugin.NewPluginManager(configPluginPath)

	// 查找所有插件
	plugins, err := pm.FindAllPlugins()
	if err != nil {
		log.Error().Err(err).Msg("Failed to find plugins")
		return
	}

	// 按来源过滤
	if sourceFilter != "" {
		plugins = filterPluginsBySource(plugins, sourceFilter)
	}

	// 如果不是全局模式，只显示当前目录的插件
	if !global {
		plugins = filterPluginsByCurrentDir(plugins)
	}

	if len(plugins) == 0 {
		log.Warn().Msg("No plugins found.")
		return
	}

	// 排序插件
	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].Source != plugins[j].Source {
			return getSourcePriority(plugins[i].Source) < getSourcePriority(plugins[j].Source)
		}
		return plugins[i].Name < plugins[j].Name
	})

	if verbose {
		displayPluginsVerbose(plugins)
	} else {
		displayPluginsSimple(plugins)
	}
}

func filterPluginsBySource(plugins []*models.PluginInfo, sourceFilter string) []*models.PluginInfo {
	var filtered []*models.PluginInfo
	for _, plugin := range plugins {
		switch sourceFilter {
		case "user", "user-home":
			if plugin.Source == models.SourceUserHome {
				filtered = append(filtered, plugin)
			}
		case "current", "current-dir":
			if plugin.Source == models.SourceCurrentDir {
				filtered = append(filtered, plugin)
			}
		case "config":
			if plugin.Source == models.SourceConfig {
				filtered = append(filtered, plugin)
			}
		}
	}
	return filtered
}

func filterPluginsByCurrentDir(plugins []*models.PluginInfo) []*models.PluginInfo {
	var filtered []*models.PluginInfo
	for _, plugin := range plugins {
		if plugin.Source == models.SourceCurrentDir {
			filtered = append(filtered, plugin)
		}
	}
	return filtered
}

func getSourcePriority(source models.PluginSource) int {
	switch source {
	case models.SourceUserHome:
		return 1
	case models.SourceCurrentDir:
		return 2
	case models.SourceConfig:
		return 3
	default:
		return 4
	}
}

func displayPluginsSimple(plugins []*models.PluginInfo) {
	// 构造表格数据
	headers := []string{"NAME", "SOURCE", "PATH"}
	var rows [][]string
	for _, plugin := range plugins {
		rows = append(rows, []string{
			plugin.GetDisplayName(),
			getSourceShortName(plugin.Source),
			plugin.Path,
		})
	}
	if err := style.PrintTable(os.Stdout, headers, rows, 0); err != nil {
		log.Error().Err(err).Msg("Failed to print plugin table")
	}
}

func displayPluginsVerbose(plugins []*models.PluginInfo) {
	headers := []string{"NAME", "SOURCE", "SIZE", "MODIFIED", "PATH"}
	var rows [][]string
	for _, plugin := range plugins {
		rows = append(rows, []string{
			plugin.GetDisplayName(),
			getSourceShortName(plugin.Source),
			formatSize(plugin.Size),
			plugin.ModTime.Format("2006-01-02 15:04"),
			plugin.Path,
		})
	}
	if err := style.PrintTable(os.Stdout, headers, rows, 0); err != nil {
		log.Error().Err(err).Msg("Failed to print plugin table")
		return
	}

	// 显示来源说明
	sources := make(map[models.PluginSource]bool)
	sourceDescriptions := make(map[string]string)
	for _, plugin := range plugins {
		if !sources[plugin.Source] {
			sources[plugin.Source] = true
			sourceDescriptions[getSourceShortName(plugin.Source)] = getSourceDescription(plugin.Source)
		}
	}
}

func getSourceShortName(source models.PluginSource) string {
	switch source {
	case models.SourceUserHome:
		return "USER"
	case models.SourceCurrentDir:
		return "LOCAL"
	case models.SourceConfig:
		return "CONFIG"
	default:
		return "UNKNOWN"
	}
}

func getSourceDescription(source models.PluginSource) string {
	switch source {
	case models.SourceUserHome:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".gocli", "plugins")
	case models.SourceCurrentDir:
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, ".gocli", "plugins")
	case models.SourceConfig:
		return "Config file specified path"
	default:
		return "Unknown source"
	}
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1fK", float64(size)/1024)
	}
	return fmt.Sprintf("%.1fM", float64(size)/(1024*1024))
}

func init() {
	rootCmd.AddCommand(pluginCmd)

	pluginCmd.AddCommand(pluginListCmd)
	pluginListCmd.Flags().BoolP("global", "g", false, "List all plugins globally installed")
	pluginListCmd.Flags().StringP("source", "s", "", "Filter by source: user, current, config")
}
