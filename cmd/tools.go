package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/configs"
	"github.com/yeisme/gocli/pkg/style"
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
  - Use --global to install configured global tools or to default single installs to $HOME/.gocli/tools.
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
					if err := batchInstallConfiguredGlobalTools(gocliCtx.Config, envFlags, v); err != nil {
						log.Error().Err(err).Msg("batch install (global) finished with errors")
					}
					return
				}
				if err := batchInstallConfiguredTools(gocliCtx.Config, envFlags, v); err != nil {
					log.Error().Err(err).Msg("batch install finished with errors")
				}
				return
			}

			// 2. 单个工具安装逻辑
			if pathFlag == "" {
				if globalFlag {
					// --global 单个安装默认路径：$HOME/.gocli/tools
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
					if bi, ok := toolsPkg.BuiltinTools[name]; ok {
						// 若配置了 CloneURL，则走 clone 构建路径；否则回退到 go install URL
						if strings.TrimSpace(bi.CloneURL) != "" {
							// 将 @version（若有）映射为 #ref
							clone := bi.CloneURL
							if ver != "" {
								ref := ver[1:] // strip leading '@'
								// 若 CloneURL 已经包含 #，则替换或追加？这里追加优先（通常 CloneURL 不带 ref）
								if strings.Contains(clone, "#") {
									// 简单策略：去掉原有片段，使用新的
									if p := strings.Index(clone, "#"); p >= 0 {
										clone = clone[:p]
									}
								}
								clone = clone + "#" + ref
							}
							// 将 clone 相关字段灌入选项（用户显式 flags 优先生效）
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
							// 清空 go install 的 spec，避免歧义
							spec = ""
							if v {
								log.Info().Msgf("mapped builtin tool %s -> clone %s (build=%s)", orig, cloneURL, toolInstallOptions.BuildMethod)
							}
						} else {
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

			opts := toolsPkg.InstallOptions{
				Spec:              spec,
				CloneURL:          cloneURL,
				MakeTarget:        makeTarget,
				Path:              pathFlag,
				Env:               envFlags,
				BinDirs:           binDirs,
				Verbose:           v,
				ReleaseBuild:      releaseBuild,
				DebugBuild:        debugBuild,
				BinaryName:        toolInstallOptions.BinaryName,
				BuildMethod:       toolInstallOptions.BuildMethod,
				BuildArgs:         toolInstallOptions.BuildArgs,
				WorkDir:           toolInstallOptions.WorkDir,
				GoreleaserConfig:  toolInstallOptions.GoreleaserConfig,
				RecurseSubmodules: toolInstallOptions.RecurseSubmodules,
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
			printInstallOutput(res.Output, err, v)
			if err != nil {
				if res.Mode != "go_install" {
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
	toolRunCmd = &cobra.Command{
		Use:   "run <tool> [args...]",
		Short: "Run a tool",
		Long: `
Run a tool by configured name or by explicit path and forward all remaining
arguments to the executed binary unchanged.

Basic usage:
  gocli tool run <tool> [args...]
  gox run <tool> [args...]

Examples:
  # Run a configured tool named "task"
  gocli tools run task
  gox run task

  # Execute a binary by path and forward flags/args
  gocli tools run task --list
  gox run task --list

Notes:
  - Use 'gocli tools list' to inspect available configured tools and their
    install paths.
  - If the first argument matches a configured tool name (case-insensitive),
    gocli will run the discovered binary for that tool.
  - If the argument looks like a path (contains path separators or is absolute),
    and the file exists, gocli will execute that path directly.
  - All flags and arguments after the tool name are forwarded verbatim to the
    invoked executable. Unknown flags are allowed so flags intended for the
    executed tool are not interpreted by cobra.
`,
		// Disable cobra flag parsing so flags after the tool name (including
		// --help) are forwarded verbatim to the executed binary. We still want
		// `gocli tools run --help` to show the run command help, so detect that
		// specific form in PreRun and print help for the run command.
		DisableFlagParsing: true,
		PreRun: func(cmd *cobra.Command, _ []string) {
			argv := os.Args
			for i := range argv {
				if argv[i] == "run" || argv[i] == "x" || argv[i] == "exec" {
					if i+1 < len(argv) {
						next := argv[i+1]
						if next == "--help" || next == "-h" {
							_ = cmd.Help()
							os.Exit(0)
						}
					}
					break
				}
			}
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

	// For the `tools run` command we want to hide all inherited/global flags
	// from its help output because this command forwards flags to the
	// executed binary. Print only usage, long text, local flags and
	// available subcommands.
	toolRunCmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		out := cmd.OutOrStdout()
		// Usage
		if cmd.Runnable() {
			fmt.Fprintf(out, "Usage: %s\n\n", cmd.UseLine())
		}
		// Long description
		if cmd.Long != "" {
			fmt.Fprintln(out, strings.TrimSpace(cmd.Long))
			fmt.Fprintln(out)
		}
		// Local flags only (omits inherited/global flags)
		if cmd.HasAvailableLocalFlags() {
			fmt.Fprintln(out, "Flags:")
			fmt.Fprint(out, cmd.LocalFlags().FlagUsages())
			fmt.Fprintln(out)
		}
		// Subcommands
		if cmd.HasAvailableSubCommands() {
			fmt.Fprintln(out, "Available Commands:")
			for _, c := range cmd.Commands() {
				if c.IsAvailableCommand() {
					fmt.Fprintf(out, "  %s\t%s\n", c.Name(), c.Short)
				}
			}
			fmt.Fprintln(out)
		}
	})

	// json flags
	toolListCmd.Flags().BoolP("json", "j", false, "Output the list of tools in JSON format")

	// flags for install
	toolInstallCmd.Flags().SortFlags = false
	toolInstallCmd.Flags().StringVarP(&toolInstallOptions.Path, "path", "p", "", "Installation output directory (effective for go install, equivalent to setting GOBIN)")
	toolInstallCmd.Flags().StringSliceVarP(&toolInstallOptions.Env, "env", "e", nil, "Additional build environment variables, e.g.: --env CGO_ENABLED=1 --env CC=clang")
	toolInstallCmd.Flags().StringVarP(&toolInstallOptions.CloneURL, "clone", "C", "", "Clone source code from a Git repository for installation, supports URL#ref syntax to specify branch/tag/commit")
	toolInstallCmd.Flags().StringVarP(&toolInstallOptions.MakeTarget, "make-target", "m", "", "Target name to execute with make in the source directory (default is make)")
	toolInstallCmd.Flags().StringSliceVarP(&toolInstallOptions.BinDirs, "dir", "d", nil, "Directory(ies) where the built binaries are output by make; repeat or separate by platform path list separator")
	toolInstallCmd.Flags().BoolVarP(&toolInstallGlobal, "global", "g", false, "Install 'tools.global' from config when used without args; when specifying a tool, default install path is $HOME/.gocli/tools")
	// build presets
	toolInstallCmd.Flags().BoolVarP(&toolInstallOptions.ReleaseBuild, "release-build", "R", false, "Install in release mode (-trimpath -ldflags '-s -w')")
	toolInstallCmd.Flags().BoolVarP(&toolInstallOptions.DebugBuild, "debug-build", "D", false, "Install in debug mode (-gcflags 'all=-N -l')")
	// binary name override (avoid conflict with --binary-name used for directories)
	toolInstallCmd.Flags().StringVarP(&toolInstallOptions.BinaryName, "binary-name", "n", "", "Override the output binary name (when determinable)")
	// clone build method and options
	toolInstallCmd.Flags().StringVarP(&toolInstallOptions.BuildMethod, "build", "b", "", "Build method when using --clone: make (default) | goreleaser")
	toolInstallCmd.Flags().StringSliceVarP(&toolInstallOptions.BuildArgs, "build-arg", "a", nil, "Extra arguments passed to the build tool (repeatable). For goreleaser, e.g. --build-arg --skip=validate")
	toolInstallCmd.Flags().StringVarP(&toolInstallOptions.WorkDir, "workdir", "w", "", "Subdirectory inside the repository to run the build in")
	toolInstallCmd.Flags().StringVar(&toolInstallOptions.GoreleaserConfig, "goreleaser-config", "", "Path to goreleaser config file (relative to repo root or workdir)")
	toolInstallCmd.Flags().BoolVarP(&toolInstallOptions.RecurseSubmodules, "recurse-submodules", "r", false, "Clone Git submodules recursively when using --clone")
}

// mustUserHome 返回用户 home 目录，若失败直接返回当前目录 (尽量不 panic 保持安装流程继续)
func mustUserHome() string {
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return "."
	}
	return h
}

// batchInstallConfiguredTools 批量安装配置文件中的 go 工具（deps -> tools.path, global -> 用户home .gocli/tools）
func batchInstallConfiguredTools(cfg *configs.Config, envFlags []string, verbose bool) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	depsPath := cfg.Tools.GoCLIToolsPath
	if strings.TrimSpace(depsPath) == "" {
		depsPath = filepath.Join(mustUserHome(), ".gocli", "tools")
	}
	globalPath := filepath.Join(mustUserHome(), ".gocli", "tools")

	// 支持 go 与 clone/git 两种类型
	totalDeps, failedDeps := installConfiguredToolsFromList(cfg.Tools.Deps, depsPath, "dep", envFlags, verbose)
	totalGlobal, failedGlobal := installConfiguredToolsFromList(cfg.Tools.Global, globalPath, "global", envFlags, verbose)

	total := totalDeps + totalGlobal
	failed := failedDeps + failedGlobal

	if total == 0 && failed == 0 {
		log.Warn().Msg("no go tools found in config for installation")
	}
	if failed > 0 {
		return fmt.Errorf("%d tool(s) failed", failed)
	}
	return nil
}

// batchInstallConfiguredGlobalTools 仅安装配置中的 global 工具到 GOROOT/bin（若可用）
func batchInstallConfiguredGlobalTools(cfg *configs.Config, envFlags []string, verbose bool) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	// 全局安装统一到用户目录 $HOME/.gocli/tools
	targetPath := filepath.Join(mustUserHome(), ".gocli", "tools")

	total, failed := installConfiguredToolsFromList(cfg.Tools.Global, targetPath, "global", envFlags, verbose)

	if total == 0 && failed == 0 {
		log.Warn().Msg("no go tools found in config.tools.global for installation")
	}
	if failed > 0 {
		return fmt.Errorf("%d tool(s) failed", failed)
	}
	return nil
}

// installGoToolsFromList 安装一组 go 类型的工具；返回成功个数与失败个数
func installConfiguredToolsFromList(list []configs.Tool, targetPath, category string, envFlags []string, verbose bool) (int, int) {
	total := 0
	failed := 0

	for _, t := range list {
		ttype := strings.ToLower(strings.TrimSpace(t.Type))
		envMerged := append([]string{}, envFlags...)
		if len(t.Env) > 0 {
			envMerged = append(envMerged, t.Env...)
		}

		switch ttype {
		case "", "go":
			// 兼容：优先 Module，其次解析 Cmd
			spec := strings.TrimSpace(t.Module)
			if spec == "" {
				if strings.TrimSpace(t.Cmd) == "" {
					// 跳过空项
					continue
				}
				s, err := parseGoInstallSpec(t.Cmd)
				if err != nil {
					failed++
					log.Error().Err(err).Msgf("parse %s tool cmd failed: %s", category, t.Cmd)
					continue
				}
				spec = s
			}
			res, err := toolsPkg.InstallTool(toolsPkg.InstallOptions{
				Spec:         spec,
				Path:         targetPath,
				Env:          envMerged,
				Verbose:      verbose,
				ReleaseBuild: t.ReleaseBuild,
				DebugBuild:   t.DebugBuild,
				BinaryName:   t.BinaryName,
			})
			printInstallOutput(res.Output, err, verbose)
			if err != nil {
				failed++
				log.Error().Err(err).Msgf("install %s tool failed: %s", category, spec)
				continue
			}
			if res.InstallDir != "" {
				log.Info().Msgf("installed %s(go): %s -> %s", category, spec, filepath.Clean(res.InstallDir))
			}
			total++

		case "clone", "git":
			if strings.TrimSpace(t.CloneURL) == "" {
				failed++
				log.Error().Msgf("install %s tool failed: missing clone url", category)
				continue
			}
			res, err := toolsPkg.InstallTool(toolsPkg.InstallOptions{
				CloneURL:          t.CloneURL,
				BuildMethod:       t.Build,
				MakeTarget:        t.MakeTarget,
				WorkDir:           t.WorkDir,
				BinDirs:           t.BinDirs,
				Env:               envMerged,
				GoreleaserConfig:  t.GoreleaserConfig,
				BinaryName:        t.BinaryName,
				RecurseSubmodules: t.RecurseSubmodules,
				ReleaseBuild:      t.ReleaseBuild,
				DebugBuild:        t.DebugBuild,
				Path:              targetPath,
				Verbose:           verbose,
			})
			printInstallOutput(res.Output, err, verbose)
			if err != nil {
				failed++
				log.Error().Err(err).Msgf("install %s tool (clone) failed: %s", category, t.CloneURL)
				continue
			}
			if res.InstallDir != "" {
				log.Info().Msgf("installed %s(clone): %s -> %s", category, t.CloneURL, filepath.Clean(res.InstallDir))
			}
			total++

		default:
			// 未知类型：跳过并提示
			log.Warn().Msgf("skip %s tool with unsupported type: %s", category, ttype)
		}
	}
	return total, failed
}

// parseGoInstallSpec 从诸如 "go install pkg@v1" 的命令中提取 spec
func parseGoInstallSpec(cmd string) (string, error) {
	fields := strings.Fields(cmd)
	if len(fields) < 3 || fields[0] != "go" || fields[1] != "install" {
		return "", fmt.Errorf("unsupported go tool cmd: %s", cmd)
	}
	for _, f := range fields[2:] {
		if strings.HasPrefix(f, "-") { // 忽略构建 flag
			continue
		}
		return f, nil
	}
	return "", fmt.Errorf("cannot find spec in: %s", cmd)
}

// printInstallOutput 按 verbose/err 级别打印安装输出
func printInstallOutput(output string, err error, verbose bool) {
	if strings.TrimSpace(output) == "" {
		return
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err != nil {
			log.Error().Msg(line)
		} else if verbose {
			log.Debug().Msg(line)
		} else {
			log.Info().Msg(line)
		}
	}
}
