// Package cmd provides command-line interface commands for gocli
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/configs"
)

var (
	configCmd = &cobra.Command{
		Use:     "config",
		Short:   "Manage gocli configuration",
		Long:    `gocli config allows you to view and manage your gocli configuration settings.`,
		Aliases: []string{"c"},
	}

	configValidateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate gocli configuration",
		Long:  `gocli config validate checks the validity of your configuration file and environment variables.`,
		Run: func(cmd *cobra.Command, _ []string) {
			// 检查配置文件加载
			err := gocliCtx.Viper.ReadInConfig()
			if err != nil {
				cmd.PrintErrf("Config file error: %v\n", err)
				os.Exit(1)
			}

			fileUsed := gocliCtx.Viper.ConfigFileUsed()

			log.Info().Msgf("Config file used: %s", fileUsed)
		},
		Aliases: []string{"check", "verify"},
	}

	configListCmd = &cobra.Command{
		Use:   "list [section]",
		Short: "List gocli configuration",
		Long: `gocli config list displays the current configuration settings.

You can specify a section to display only that part of the configuration:
  - app: Application settings
  - env: Environment settings
  - log: Logging settings

Examples:
  gocli config list                    # Show all configuration (viper raw data)
  gocli config list --all              # Show all configuration with defaults
  gocli config list app                # Show only app settings
  gocli config list --format yaml      # Output in YAML format
  gocli config list --format json      # Output in JSON format
  gocli config list --yaml             # Output in YAML format (shorthand)
  gocli config list app --all --json   # Show app config with defaults in JSON`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			section := ""
			if len(args) > 0 {
				section = args[0]
			}

			// 确定输出格式
			format := configs.GetOutputFormatFromFlags(cmd)

			// 检查是否显示完整配置（包含默认值）
			showAll, _ := cmd.Flags().GetBool("all")

			// 获取配置数据
			data, err := configs.GetConfigSection(gocliCtx.Viper, section, showAll)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting config section: %v\n", err)
				os.Exit(1)
			}

			// 输出配置
			if err := configs.OutputData(data, format); err != nil {
				fmt.Fprintf(os.Stderr, "Error displaying config: %v\n", err)
				os.Exit(1)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(
		configListCmd,
		configValidateCmd,
	)

	// 添加输出格式标志
	configListCmd.Flags().StringP("format", "f", "", fmt.Sprintf("Output format (%s)", strings.Join(configs.ValidFormats(), ", ")))
	configListCmd.Flags().Bool("yaml", false, "Output in YAML format")
	configListCmd.Flags().Bool("json", false, "Output in JSON format")
	configListCmd.Flags().Bool("toml", false, "Output in TOML format")
	configListCmd.Flags().Bool("text", false, "Output in plain text format")
	configListCmd.Flags().BoolP("all", "a", false, "Show complete configuration with defaults (processed struct)")
}
