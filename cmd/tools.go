package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
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
		Long:  `gocli tools install allows you to install a specific tool or utility for your Go development environment.`,
		Example: strings.TrimSpace(`
  # Example 1: Install the latest version of a tool
  # This is the most common usage, installing directly from a Go module path.
  gocli tools install github.com/golangci/golangci-lint/cmd/golangci-lint

  # Example 2: Install a specific version of a tool using the '@' symbol
  # Append '@version' to the tool path to get a stable release.
  gocli tools install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1

  # Example 3: Install the tool to a custom directory
  # Use the --path flag to specify where the binary should be placed.
  gocli tools install --path ./bin github.com/golangci/golangci-lint/cmd/golangci-lint

  # Example 4: Install from a local source path
  # If you have the tool's source code locally, you can install it by specifying its path.
  gocli tools install ./internal/my-local-tool

  # Example 5: Clone from a Git repository and compile using make
  # For complex projects, you can clone the repository and then run a specific target in the 'make' command to build.
  gocli tools install --clone https://github.com/projectdiscovery/httpx.git --make-target build

  # Example 6: Clone a specific Git tag or branch and install
  # Use '#tag' or '#branch' after the repository URL to specify the version.
  gocli tools install --clone https://github.com/docker/compose.git#v2.39.2 --make-target build

  # Example 7: Set build environment variables for tools that require CGo
  # Use the --env flag to pass environment variables required at compile time.
  gocli tools install --env "CGO_ENABLED=1" github.com/example/cgo-tool

	# Example 8: Install in release mode (preconfigured flags)
	# Equivalent to adding: -trimpath -ldflags="-s -w" (and -v when --verbose)
	gocli tools install --release-build github.com/golangci/golangci-lint/cmd/golangci-lint

	# Example 9: Install in debug mode (preconfigured flags)
	# Equivalent to adding: -gcflags=all=-N -l (and -v when --verbose)
	gocli tools install --debug-build github.com/golangci/golangci-lint/cmd/golangci-lint

	# Example 10: Clone + make with custom output directories
	# When the Makefile builds binaries into custom folders, use --bin to tell gocli where to pick them up.
	# You can provide multiple directories by repeating --bin or using the platform path list separator.
	gocli tools install --clone https://github.com/docker/compose.git#v2.39.2 --make-target build --bin ./cmd/build --bin ./bin
`),
		Run: func(cmd *cobra.Command, args []string) {
			cloneURL, _ := cmd.Flags().GetString("clone")
			makeTarget, _ := cmd.Flags().GetString("make-target")
			pathFlag, _ := cmd.Flags().GetString("path")
			envFlags, _ := cmd.Flags().GetStringSlice("env")
			binDirs, _ := cmd.Flags().GetStringSlice("bin")
			releaseBuild, _ := cmd.Flags().GetBool("release-build")
			debugBuild, _ := cmd.Flags().GetBool("debug-build")

			if pathFlag == "" {
				pathFlag = gocliCtx.Config.Tools.GoCLIToolsPath
			}

			v := verbose
			var spec string
			if len(args) > 0 {
				spec = args[0]
			}

			opts := toolsPkg.InstallOptions{
				Spec:         spec,
				CloneURL:     cloneURL,
				MakeTarget:   makeTarget,
				Path:         pathFlag,
				Env:          envFlags,
				BinDirs:      binDirs,
				Verbose:      v,
				ReleaseBuild: releaseBuild,
				DebugBuild:   debugBuild,
			}

			if opts.CloneURL == "" && opts.Spec == "" {
				log.Error().Msg("missing tool spec. Provide a module path or use --clone.")
				return
			}

			var sp *style.Spinner
			if !v && !quiet {
				label := "Installing"
				if opts.CloneURL != "" {
					label = "Cloning and building"
				}
				sp = style.NewSpinner(cmd.OutOrStdout(), label)
				sp.Start()
			}

			res, err := toolsPkg.InstallTool(opts)
			if sp != nil {
				sp.Stop()
			}

			// 逐行输出
			for line := range strings.SplitSeq(res.Output, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if err != nil {
					log.Error().Msg(line)
				} else if v {
					log.Debug().Msg(line)
				} else {
					log.Info().Msg(line)
				}
			}

			if err != nil {
				if res.Mode == "clone_make" {
					log.Error().Err(err).Msg("install via clone failed")
				} else {
					log.Error().Err(err).Msg("go install failed")
				}
				return
			}

			if res.InstallDir != "" {
				log.Info().Msgf("installed to: %s", filepath.Clean(res.InstallDir))
			}
			if res.ProbableInstallDir != "" && res.InstallDir == "" {
				log.Info().Msgf("probable install dir: %s", filepath.Clean(res.ProbableInstallDir))
			}
		},
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

	// flags for install
	toolInstallCmd.Flags().StringP("path", "p", "", "Installation output directory (effective for go install, equivalent to setting GOBIN)")
	toolInstallCmd.Flags().StringSlice("env", nil, "Additional build environment variables, e.g.: --env CGO_ENABLED=1 --env CC=clang")
	toolInstallCmd.Flags().String("clone", "", "Clone source code from a Git repository for installation, supports URL#ref syntax to specify branch/tag/commit")
	toolInstallCmd.Flags().String("make-target", "", "Target name to execute with make in the source directory (default is make)")
	toolInstallCmd.Flags().StringSlice("bin", nil, "Directory(ies) where the built binaries are output by make; repeat or separate by platform path list separator")
	// build presets
	toolInstallCmd.Flags().Bool("release-build", false, "Install in release mode (-trimpath -ldflags '-s -w')")
	toolInstallCmd.Flags().Bool("debug-build", false, "Install in debug mode (-gcflags 'all=-N -l')")
}
