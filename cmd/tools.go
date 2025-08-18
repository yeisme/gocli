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
		Long: `gocli tools list displays all available tools that can be used with gocli.

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

			if v {
				headers := []string{"name", "source", "size", "modified", "path"}
				rows := make([][]string, 0, len(tools))
				for _, t := range tools {
					rows = append(rows, []string{
						t.Name,
						string(t.Source),
						formatSize(t.Size),
						t.ModTime.Format("2006-01-02 15:04"),
						t.Path,
					})
				}
				if err := style.PrintTable(cmd.OutOrStdout(), headers, rows, 0); err != nil {
					log.Error().Err(err).Msg("failed to print tools list in table format")
				}
			} else {
				headers := []string{"name", "source", "path"}
				rows := make([][]string, 0, len(tools))
				for _, t := range tools {
					rows = append(rows, []string{t.Name, string(t.Source), t.Path})
				}
				if err := style.PrintTable(cmd.OutOrStdout(), headers, rows, 0); err != nil {
					log.Error().Err(err).Msg("failed to print tools list in table format")
				}
			}
		},
	}

	toolInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install a tool",
		Long: `gocli tools install allows you to install a specific tool or utility for your Go development environment.

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
						mapped := bi.URL + ver
						spec = mapped
						if v {
							log.Info().Msgf("mapped builtin tool %s -> %s", orig, spec)
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
			printInstallOutput(res.Output, err, v)
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

	// json flags
	toolListCmd.Flags().BoolP("json", "j", false, "Output the list of tools in JSON format")

	// flags for install
	toolInstallCmd.Flags().SortFlags = false
	toolInstallCmd.Flags().StringVarP(&toolInstallOptions.Path, "path", "p", "", "Installation output directory (effective for go install, equivalent to setting GOBIN)")
	toolInstallCmd.Flags().StringSliceVar(&toolInstallOptions.Env, "env", nil, "Additional build environment variables, e.g.: --env CGO_ENABLED=1 --env CC=clang")
	toolInstallCmd.Flags().StringVar(&toolInstallOptions.CloneURL, "clone", "", "Clone source code from a Git repository for installation, supports URL#ref syntax to specify branch/tag/commit")
	toolInstallCmd.Flags().StringVar(&toolInstallOptions.MakeTarget, "make-target", "", "Target name to execute with make in the source directory (default is make)")
	toolInstallCmd.Flags().StringSliceVar(&toolInstallOptions.BinDirs, "bin", nil, "Directory(ies) where the built binaries are output by make; repeat or separate by platform path list separator")
	toolInstallCmd.Flags().BoolVarP(&toolInstallGlobal, "global", "g", false, "Install 'tools.global' from config when used without args; when specifying a tool, default install path is $HOME/.gocli/tools")
	// build presets
	toolInstallCmd.Flags().BoolVar(&toolInstallOptions.ReleaseBuild, "release-build", false, "Install in release mode (-trimpath -ldflags '-s -w')")
	toolInstallCmd.Flags().BoolVar(&toolInstallOptions.DebugBuild, "debug-build", false, "Install in debug mode (-gcflags 'all=-N -l')")
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

	// 仅对 go 类型保留扩展性：分别安装 deps 和 global 列表
	totalDeps, failedDeps := installGoToolsFromList(cfg.Tools.Deps, depsPath, "dep", envFlags, verbose)
	totalGlobal, failedGlobal := installGoToolsFromList(cfg.Tools.Global, globalPath, "global", envFlags, verbose)

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

	total, failed := installGoToolsFromList(cfg.Tools.Global, targetPath, "global", envFlags, verbose)

	if total == 0 && failed == 0 {
		log.Warn().Msg("no go tools found in config.tools.global for installation")
	}
	if failed > 0 {
		return fmt.Errorf("%d tool(s) failed", failed)
	}
	return nil
}

// installGoToolsFromList 安装一组 go 类型的工具；返回成功个数与失败个数
func installGoToolsFromList(list []configs.Tool, targetPath, category string, envFlags []string, verbose bool) (int, int) {
	total := 0
	failed := 0

	for _, t := range list {
		if strings.ToLower(t.Type) != "go" {
			continue
		}
		spec, err := parseGoInstallSpec(t.Cmd)
		if err != nil {
			failed++
			log.Error().Err(err).Msgf("parse %s tool cmd failed: %s", category, t.Cmd)
			continue
		}
		res, err := toolsPkg.InstallTool(toolsPkg.InstallOptions{Spec: spec, Path: targetPath, Env: envFlags, Verbose: verbose})
		printInstallOutput(res.Output, err, verbose)
		if err != nil {
			failed++
			log.Error().Err(err).Msgf("install %s tool failed: %s", category, spec)
			continue
		}
		if res.InstallDir != "" {
			log.Info().Msgf("installed %s: %s -> %s", category, spec, filepath.Clean(res.InstallDir))
		}
		total++
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
