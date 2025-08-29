package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	toolsPkg "github.com/yeisme/gocli/pkg/tools"
)

var (
	toolInstallOptions toolsPkg.InstallOptions
	toolInstallGlobal  bool

	toolsCmd = &cobra.Command{
		Use:     "tools",
		Short:   "Tools Management for gocli",
		Long:    `gocli tools allows you to manage various tools and utilities that enhance your golang development experience.`,
		Aliases: []string{"tool", "t"},
	}

	toolListCmd = &cobra.Command{
		Use:   "list",
		Short: "List available tools",
		Long: `
gocli tools list displays all available tools that can be used with gocli.

Examples:
  gocli tools list
  gocli tools list --json
`,
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

			if err := toolsPkg.PrintToolsTable(cmd.OutOrStdout(), tools, v); err != nil {
				log.Error().Err(err).Msg("failed to print tools list in table format")
			}
		},
	}

	toolInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install a tool",
		Long: `
Install a Go tool either via module path (go install style) or by cloning a repository and building it.

Basic usage:
  gocli tools install <module-or-local-path>
  gocli tools install --clone <git-url[#ref]> [flags]

Examples:
  # 1. Install the latest version of a tool
  # This is the most common usage, installing directly from a Go module path.
  gocli tools install github.com/golangci/golangci-lint/cmd/golangci-lint

  # 2. Install a specific version of a tool using the '@' symbol
  # Append '@version' to the tool path to get a stable release.
  gocli tools install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1

  # 3. Install the tool to a custom directory
  # Use the --path flag to specify where the binary should be placed.
  gocli tools install --path ./bin github.com/golangci/golangci-lint/cmd/golangci-lint

  # 4. Install from a local source path
  # If you have the tool's source code locally, you can install it by specifying its path.
  gocli tools install ./internal/my-local-tool

  # 5. Clone from a Git repository and compile using make
  # For complex projects, you can clone the repository and then run a specific target in the 'make' command to build.
  gocli tools install --clone https://github.com/projectdiscovery/httpx.git --make-target build

  # 6. Clone a specific Git tag or branch and install
  # Use '#tag' or '#branch' after the repository URL to specify the version.
  gocli tools install --clone https://github.com/docker/compose.git#v2.39.2 --make-target build

  # 7. Set build environment variables for tools that require CGo
  # Use the --env flag to pass environment variables required at compile time.
  gocli tools install --env "CGO_ENABLED=1" github.com/example/cgo-tool

  # 8. Install in release mode (preconfigured flags)
  # Equivalent to adding: -trimpath -ldflags="-s -w" (and -v when --verbose)
  gocli tools install --release-build github.com/golangci/golangci-lint/cmd/golangci-lint

  # 9. Install in debug mode (preconfigured flags)
  # Equivalent to adding: -gcflags=all=-N -l (and -v when --verbose)
  gocli tools install --debug-build github.com/golangci/golangci-lint/cmd/golangci-lint

  # 10. Clone + make with custom output directories
  # When the Makefile builds binaries into custom folders, use --bin to tell gocli where to pick them up.
  # You can provide multiple directories by repeating --bin or using the platform path list separator.
  gocli tools install --clone https://github.com/docker/compose.git#v2.39.2 --make-target build --bin ./cmd/build --bin ./bin

  # 11. Clone + goreleaser build (auto collect dist/* binaries)
  gocli tools install --clone https://github.com/owner/repo.git#v1.2.3 --build goreleaser --workdir ./cmd/app --binary-name app

  # 12. Clone + goreleaser with custom config and extra flags
  gocli tools install --clone https://github.com/owner/repo.git --build goreleaser --goreleaser-config .goreleaser.yml --build-arg --skip=validate

Notes:
  - When invoked without arguments and without --clone, gocli installs tools configured in your config file.
	- Use --global to install configured global tools or to default single installs to ~/.gocli/tools.
  - --release-build and --debug-build are mutually exclusive.
  - When a short builtin tool name is provided (no path separator), gocli may map it to a configured module or clone URL from builtin tool mappings.
  - Do not specify both a module/local spec and --clone at the same time; they are mutually exclusive.
`,

		Run: func(cmd *cobra.Command, args []string) {
			cloneURL := toolInstallOptions.CloneURL
			makeTarget := toolInstallOptions.MakeTarget
			pathFlag := toolInstallOptions.Path
			envFlags := toolInstallOptions.Env
			binDirs := toolInstallOptions.BinDirs
			releaseBuild := toolInstallOptions.ReleaseBuild
			debugBuild := toolInstallOptions.DebugBuild
			globalFlag := toolInstallGlobal

			v := verbose

			// 校验互斥选项
			if releaseBuild && debugBuild {
				log.Error().Msg("--release-build and --debug-build cannot be used together")
				return
			}

			// 1. 无参数 && 无 --clone -> 批量安装配置中工具
			if cloneURL == "" && len(args) == 0 {
				if globalFlag {
					if err := toolsPkg.BatchInstallConfiguredGlobalTools(gocliCtx.Config, envFlags, v); err != nil {
						log.Error().Err(err).Msg("batch install (global) finished with errors")
					}
					return
				}
				if err := toolsPkg.BatchInstallConfiguredTools(gocliCtx.Config, envFlags, v); err != nil {
					log.Error().Err(err).Msg("batch install finished with errors")
				}
				return
			}

			// 2. 单个工具安装逻辑
			if pathFlag == "" {
				if globalFlag {
					// --global 单个安装默认路径：~/.gocli/tools
					pathFlag = filepath.Join(mustUserHome(), ".gocli", "tools")
					if v {
						log.Info().Msgf("--global selected: default install path -> %s", filepath.Clean(pathFlag))
					}
				} else {
					pathFlag = gocliCtx.Config.Tools.GoCLIToolsPath
				}
			}

			var spec string
			if len(args) > 0 {
				spec = args[0]
				// 支持内置工具短名：例如 `gocli tools install golangci-lint` -> 使用 pkg/tools.BuiltinTools 中配置的 URL
				// 仅当输入不包含路径分隔符时才尝试映射（避免覆盖模块路径或本地路径）
				if !strings.Contains(spec, "/") && !strings.Contains(spec, "\\") {
					orig := spec
					name := orig
					ver := ""
					if i := strings.Index(orig, "@"); i > 0 {
						name = orig[:i]
						ver = orig[i:] // keep leading '@'
					}
					// 1) 精确 key 匹配
					bi := toolsPkg.SearchTools(name, gocliCtx.Config.Tools.ToolsConfigDir)
					// 2) 若未命中，尝试按 Name 字段匹配（如 key 为 docker-compose, name 为 compose）
					if bi == nil {
						for _, t := range toolsPkg.BuiltinTools {
							if t.Name == name {
								tt := t
								bi = &tt
								break
							}
						}
					}
					if bi != nil {
						if strings.TrimSpace(bi.CloneURL) != "" { // clone 模式
							clone := bi.CloneURL
							if ver != "" { // 将 @version（若有）映射为 #ref
								ref := ver[1:]                              // strip '@'
								if p := strings.Index(clone, "#"); p >= 0 { // 去掉旧 ref
									clone = clone[:p]
								}
								clone = clone + "#" + ref
							}
							cloneURL = clone
							if toolInstallOptions.BuildMethod == "" && strings.TrimSpace(bi.Build) != "" {
								toolInstallOptions.BuildMethod = bi.Build
							}
							if toolInstallOptions.MakeTarget == "" && strings.TrimSpace(bi.MakeTarget) != "" {
								toolInstallOptions.MakeTarget = bi.MakeTarget
							}
							if toolInstallOptions.WorkDir == "" && strings.TrimSpace(bi.WorkDir) != "" {
								toolInstallOptions.WorkDir = bi.WorkDir
							}
							if len(bi.BinDirs) > 0 {
								binDirs = append(binDirs, bi.BinDirs...)
							}
							if len(bi.Env) > 0 {
								envFlags = append(envFlags, bi.Env...)
							}
							if toolInstallOptions.GoreleaserConfig == "" && strings.TrimSpace(bi.GoreleaserConfig) != "" {
								toolInstallOptions.GoreleaserConfig = bi.GoreleaserConfig
							}
							if bi.BinaryName != "" && toolInstallOptions.BinaryName == "" {
								toolInstallOptions.BinaryName = bi.BinaryName
							}
							spec = "" // 走 clone 路径
							if v {
								log.Info().Msgf("mapped builtin tool %s -> clone %s (build=%s)", orig, cloneURL, toolInstallOptions.BuildMethod)
							}
						} else { // go install 模式
							mapped := bi.URL + ver
							spec = mapped
							if bi.BinaryName != "" && toolInstallOptions.BinaryName == "" {
								toolInstallOptions.BinaryName = bi.BinaryName
							}
							if v {
								log.Info().Msgf("mapped builtin tool %s -> %s", orig, spec)
							}
						}
					}
				}
			}

			// 同时给出 --clone 与 spec -> 不允许，避免歧义
			if cloneURL != "" && spec != "" {
				log.Error().Msg("please specify either a module/local path or --clone, not both")
				return
			}

			installOpts := toolsPkg.InstallCommandOptions{
				Args: args,
				InstallOptions: toolsPkg.InstallOptions{
					CloneURL:          cloneURL,
					MakeTarget:        makeTarget,
					Path:              pathFlag,
					Env:               envFlags,
					BinDirs:           binDirs,
					ReleaseBuild:      releaseBuild,
					DebugBuild:        debugBuild,
					BinaryName:        toolInstallOptions.BinaryName,
					BuildMethod:       toolInstallOptions.BuildMethod,
					BuildArgs:         toolInstallOptions.BuildArgs,
					WorkDir:           toolInstallOptions.WorkDir,
					GoreleaserConfig:  toolInstallOptions.GoreleaserConfig,
					RecurseSubmodules: toolInstallOptions.RecurseSubmodules,
					Force:             toolInstallOptions.Force,
					Verbose:           v,
				},
				Global:         globalFlag,
				Quiet:          quiet,
				GoCLIToolsPath: gocliCtx.Config.Tools.GoCLIToolsPath,
				ToolsConfigDir: gocliCtx.Config.Tools.ToolsConfigDir,
			}

			if err := toolsPkg.ExecuteInstallCommand(installOpts, cmd.OutOrStdout()); err != nil {
				log.Error().Err(err).Msg("install failed")
				return
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
		Use:   "search [query]",
		Short: "Search for a tool",
		Long: `
Search builtin (and user-defined) tools.
Behaviour change:
  - With a query argument: perform fuzzy search (non-interactive) using github.com/lithammer/fuzzysearch, output results directly.
  - Without any argument: enter interactive selection (fuzzy finder) to pick a tool, then print it.
`,
		Run: func(cmd *cobra.Command, args []string) {
			// format flags
			fmtFlag, _ := cmd.Flags().GetString("format")
			listJSON, _ := cmd.Flags().GetBool("json")
			listYAML, _ := cmd.Flags().GetBool("yaml")
			listTable, _ := cmd.Flags().GetBool("table")

			setCount := 0
			if cmd.Flags().Changed("format") {
				setCount++
			}
			if listJSON {
				setCount++
			}
			if listYAML {
				setCount++
			}
			if listTable {
				setCount++
			}
			if setCount > 1 {
				cmd.PrintErrf("only one of --format, --json, --yaml, --table may be specified\n")
				return
			}
			if listJSON {
				fmtFlag = "json"
			} else if listYAML {
				fmtFlag = "yaml"
			} else if listTable {
				fmtFlag = "table"
			}
			if fmtFlag == "" {
				fmtFlag = "table"
			}

			out := cmd.OutOrStdout()

			query := ""
			if len(args) > 0 {
				query = args[0]
			}

			searchOpts := toolsPkg.SearchCommandOptions{
				Query:     query,
				Format:    fmtFlag,
				JSON:      listJSON,
				YAML:      listYAML,
				Table:     listTable,
				ConfigDir: gocliCtx.Config.Tools.ToolsConfigDir,
			}

			if err := toolsPkg.ExecuteSearchCommand(searchOpts, out); err != nil {
				cmd.PrintErrf("search failed: %v\n", err)
				return
			}
		},
		Aliases: []string{"s"},
		PreRun: func(_ *cobra.Command, _ []string) {
			// load user tools then list all
			for _, p := range gocliCtx.Config.Tools.ToolsConfigDir {
				if err := toolsPkg.LoadUserTools(p); err != nil {
					log.Warn().Err(err).Msgf("failed to load user tools from: %s", p)
				} else {
					log.Debug().Msgf("loaded user tools from: %s", p)
				}
			}
		},
	}
	toolRunCmd = &cobra.Command{
		Use:   "run <tool> [args...]",
		Short: toolsPkg.ToolsRunMsg.Short,
		Long:  toolsPkg.ToolsRunMsg.Long,
		// Disable cobra flag parsing so flags after the tool name (including
		// --help) are forwarded verbatim to the executed binary. We still want
		// `gocli tools run --help` to show the run command help, so detect that
		// specific form in PreRun and print help for the run command.
		DisableFlagParsing: true,
		PreRun: func(cmd *cobra.Command, _ []string) {
			toolsPkg.ShowRunHelpIfRequested(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			gocliToolsPath := gocliCtx.Config.Tools.GoCLIToolsPath
			if err := toolsPkg.ExecuteToolRun(args, cmd.OutOrStdout(), verbose, gocliToolsPath); err != nil {
				log.Error().Err(err).Msg("failed to execute tool")
			}
		},
		// allow unknown flags so that flags intended for the executed tool
		// are not treated as errors by cobra and can be forwarded.
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		Aliases:            []string{"x", "exec"},
	}
)

// addListFlags registers flags for the `tools list` command.
func addToolsListFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("json", "j", false, "Output the list of tools in JSON format")
}

// addToolsInstallFlags registers flags for the `tools install` command.
func addToolsInstallFlags(cmd *cobra.Command, opts *toolsPkg.InstallOptions, global *bool) {
	cmd.Flags().SortFlags = false
	cmd.Flags().StringVarP(&opts.Path, "path", "p", "", "Installation output directory (effective for go install, equivalent to setting GOBIN)")
	cmd.Flags().StringSliceVarP(&opts.Env, "env", "e", nil, "Additional build environment variables, e.g.: --env CGO_ENABLED=1 --env CC=clang")
	cmd.Flags().StringVarP(&opts.CloneURL, "clone", "C", "", "Clone source code from a Git repository for installation, supports URL#ref syntax to specify branch/tag/commit")
	cmd.Flags().StringVarP(&opts.MakeTarget, "make-target", "m", "", "Target name to execute with make in the source directory (default is make)")
	cmd.Flags().StringSliceVarP(&opts.BinDirs, "dir", "d", nil, "Directory(ies) where the built binaries are output by make; repeat or separate by platform path list separator")
	cmd.Flags().BoolVarP(global, "global", "g", false, "Install 'tools.global' from config when used without args; when specifying a tool, default install path is ~/.gocli/tools")
	// build presets
	cmd.Flags().BoolVarP(&opts.ReleaseBuild, "release-build", "R", false, "Install in release mode (-trimpath -ldflags '-s -w')")
	cmd.Flags().BoolVarP(&opts.DebugBuild, "debug-build", "D", false, "Install in debug mode (-gcflags 'all=-N -l')")
	// binary name override (avoid conflict with --binary-name used for directories)
	cmd.Flags().StringVarP(&opts.BinaryName, "binary-name", "n", "", "Override the output binary name (when determinable)")
	// clone build method and options
	cmd.Flags().StringVarP(&opts.BuildMethod, "build", "b", "", "Build method when using --clone: make (default) | goreleaser")
	cmd.Flags().StringSliceVarP(&opts.BuildArgs, "build-arg", "a", nil, "Extra arguments passed to the build tool (repeatable). For goreleaser, e.g. --build-arg --skip=validate")
	cmd.Flags().StringVarP(&opts.WorkDir, "workdir", "w", "", "Subdirectory inside the repository to run the build in")
	cmd.Flags().StringVar(&opts.GoreleaserConfig, "goreleaser-config", "", "Path to goreleaser config file (relative to repo root or workdir)")
	cmd.Flags().BoolVarP(&opts.RecurseSubmodules, "recurse-submodules", "r", false, "Clone Git submodules recursively when using --clone")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Force reinstallation even if the tool already exists (overwrites existing installation)")
}

// addToolsSearchFlags registers flags for the `tools search` command.
func addToolsSearchFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("format", "f", "table", "Output format: json|yaml|table (default table)")
	cmd.Flags().BoolP("json", "j", false, "Output the search result in JSON format")
	cmd.Flags().BoolP("yaml", "y", false, "Output the search result in YAML format (overrides -f)")
	cmd.Flags().BoolP("table", "t", false, "Output the search result in table format (default)")
}

// addToolsRunFlags registers flags for the `tools run` command.
// Currently the run command intentionally disables flag parsing, but we
// keep this function as a placeholder to match the project-style organization.
func addToolsRunFlags(_ *cobra.Command) {
}

func mustUserHome() string {
	h, _ := os.UserHomeDir()
	return h
}

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
		toolRunCmd,
	)

	// Reuse the common run-style help formatter so gox and tools run share help
	// output and behavior.
	toolsPkg.SetRunHelpFunc(toolRunCmd)

	// register flags via helper functions (extracted for clarity / reuse)
	addToolsListFlags(toolListCmd)
	addToolsInstallFlags(toolInstallCmd, &toolInstallOptions, &toolInstallGlobal)
	addToolsSearchFlags(toolSearchCmd)
	addToolsRunFlags(toolRunCmd)
}
