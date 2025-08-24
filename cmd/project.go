package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/project"
	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/tools"
	"github.com/yeisme/gocli/pkg/utils/doc"
)

var (
	buildOptions  project.BuildRunOptions
	runOptions    project.BuildRunOptions
	infoOptions   project.InfoOptions
	fmtOptions    project.FmtOptions
	lintOptions   project.LintOptions
	listOptions   project.ListOptions
	updateOptions project.UpdateOptions
	depsOptions   project.DepsOptions
	docOptions    project.DocOptions
	initOptions   project.InitOptions

	projectCmd = &cobra.Command{
		Use:     "project",
		Short:   "Manage Go projects",
		Long:    `gocli project allows you to manage your Go projects, including creating, building, and running them.`,
		Aliases: []string{"p", "pj", "go"},
	}

	projectInitCmd = &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize a new Go project",
		Long:  `Initialize a new Go project with the necessary files and directories.`,
		Example: strings.TrimSpace(`
  gocli project init myproject
  gocli project init .
  gocli project init --name myproject --path /path/to/project
  gocli project init --name myproject --path /path/to/project --template gin-gorm
`),
		Run: func(cmd *cobra.Command, args []string) {
			if err := project.ExecuteInitCommand(gocliCtx, args, initOptions, cmd.OutOrStdout()); err != nil {
				// 如果是 ExecError（包含 stderr），直接把格式化后的错误作为消息打印，避免 zerolog 将换行转义
				if ee, ok := err.(*tools.ExecError); ok {
					log.Error().Msgf("failed to initialize project: %s", ee.Error())
				} else {
					log.Error().Err(err).Msg("failed to initialize project")
				}
			}
		},
	}
	projectBuildCmd = &cobra.Command{
		Use:   "build [args...] [packages]",
		Short: "Build the Go project",
		Long: strings.TrimSpace(`
gocli project build compiles Go packages (a superset wrapper of 'go build').

Basic usage:
  When no package/path is specified, the current directory is used.
  You may pass a directory, file, or pattern (e.g. ./..., ./cmd/server).

Examples:
  # 1. Build current module (similar to 'go build')
  gocli project build

  # 2. Build all sub-packages
  gocli project build ./...

  # 3. Build with custom output binary name
  gocli project build -o bin/myapp ./cmd/server

  # 4. Change directory before build (like 'cd examples/web && go build')
  gocli project build -C ./examples/web ./cmd/server

  # 5. Force full rebuild and set parallelism to 4
  gocli project build -a -p 4 ./cmd/server

  # 6. Enable race detector
  gocli project build --race ./cmd/server

  # 7. Memory / Address sanitizers (platform / compiler dependent)
  gocli project build --msan ./cmd/server
  gocli project build --asan ./cmd/server

  # 8. Optimization / size flags (strip symbols + remove file paths)
  gocli project build --tags "sqlite,json" --ldflags "-s -w" --trimpath

  # 9. Disable optimizations & inlining for debugging
  gocli project build --gcflags "all=-N -l" ./cmd/cli

  # 10. Specify build mode (e.g. position independent executable)
  gocli project build --buildmode=pie ./cmd/cli

  # 11. Use vendor mode for dependencies
  gocli project build --mod=vendor ./...

  # 12. Coverage instrumentation (mostly used with tests; here for demo)
  gocli project build --cover --covermode=atomic --coverpkg=./... ./...

  # 13. Keep temporary work directory (inspect intermediate files)
  gocli project build --work -x ./cmd/server

  # 14. Quick release-style build (smaller binary, stripped info)
  gocli project build --release-mode ./cmd/cli

  # 15. Debug-style build (no optimizations, full symbols)
  gocli project build --debug-mode ./cmd/cli

Advanced notes:
  - Most flags map directly to 'go build' counterparts (asmflags/gcflags/ldflags...).
  - --release-mode / --debug-mode are opinionated presets combining common flags.
  - Can be combined with --hot-reload (more commonly used under 'run').
`),
		Run: func(cmd *cobra.Command, args []string) {
			buildOptions.V = gocliCtx.Config.App.Verbose
			if err := project.ExecuteBuildCommand(gocliCtx, buildOptions, args); err != nil {
				cmd.PrintErrf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	projectRunCmd = &cobra.Command{
		Use:   "run [args...] [packages]",
		Short: "Run the Go project",
		Long: strings.TrimSpace(`
gocli project run builds then runs one (or multiple) main entrypoints (main package / main.go).

Core capabilities:
	- Automatically triggers a build with the provided flags before execution.
	- Supports changing working directory via -C.
	- Supports hot reloading (--hot-reload / -r) to auto rebuild & restart on file changes.

Examples:
  # 1. Run the main package in the current directory
  gocli project run

  # 2. Run a specific main file
  gocli project run main.go

  # 3. Run a specific entry directory
  gocli project run ./cmd/server

  # Parallelism & build control:
  # 4. Set parallelism during build
  gocli project run -p 2 ./cmd/server

  # Build tags & module mode:
  # 5. Use build tags (e.g. dev)
  gocli project run --tags "dev" ./cmd/server
  # 6. Set module download mode (readonly/vendor/mod)
  gocli project run --mod=mod ./cmd/server

  # Debugging & performance:
  # 7. Debug mode (disable opt, keep symbols)
  gocli project run --debug-mode ./cmd/server
  # 8. Race detector
  gocli project run --race ./cmd/server

  # Hot reload:
  # 9. Enable hot reload (rebuild & restart on change)
  gocli project run -r ./cmd/server
  # 10. Hot reload without respecting .gitignore
  gocli project run -r --no-gitignore ./cmd/server

  # Additional tips:
  - Hot reload is for local dev; for production prefer a static build + external supervisor.
  - --release-mode may also be used here to emulate production flags for a quick run.
  - Use -n / --dry-run to only print the underlying commands.
`),
		Run: func(cmd *cobra.Command, args []string) {
			runOptions.V = gocliCtx.Config.App.Verbose
			if err := project.ExecuteRunCommand(gocliCtx, runOptions, args); err != nil {
				cmd.PrintErrf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	projectListCmd = &cobra.Command{
		Use:   "list [flags] [patterns]",
		Short: "List Go packages (wrapper around 'go list')",
		Long: `gocli project list lists Go packages under the current module.

By default it expands to './...' to list all packages.
Patterns can be provided (e.g. ./cmd/..., ./pkg/utils, ./... ).

Examples:
  # List all packages
  gocli project list

  # List specific pattern
  gocli project list ./cmd/
  gocli project list ./...

  # Include test packages
  gocli project list --test

  # JSON output
  gocli project list --json > pkgs.json

  # Verbose (show total count)
  gocli project list -v
`,
		Run: func(cmd *cobra.Command, args []string) {
			// Execute list
			var b strings.Builder
			if err := project.RunList(listOptions, &b, args); err != nil {
				log.Error().Err(err).Msg("failed to run project list")
				os.Exit(1)
			}
			output := b.String()
			// JSON: pass-through
			if listOptions.JSON {
				_ = style.PrintJSONLine(cmd.OutOrStdout(), output)
				return
			}
			trimmed := strings.TrimSpace(output)
			if trimmed != "" {
				lines := strings.Split(trimmed, "\n")
				pkgs := make([]string, 0, len(lines))
				for _, l := range lines {
					l = strings.TrimSpace(l)
					if l == "" {
						continue
					}
					pkgs = append(pkgs, l)
				}
				if len(pkgs) > 0 {
					_ = style.PrintPackageList(cmd.OutOrStdout(), pkgs)
				}
				if verbose && !quiet {
					cmd.Printf("Total: %d packages\n", len(pkgs))
				}
			} else if verbose && !quiet {
				cmd.Println("No packages found")
			}
		},
	}
	projectInfoCmd = &cobra.Command{
		Use:   "info [flags]",
		Short: "Show information about the Go project",
		Long: `gocli project info provides detailed information about the Go project, including file statistics and language breakdowns.

Examples:
  gocli project info

  # Specify a directory and output as JSON
  gocli project info ./ --json

  # Only include specified patterns (can be repeated)
  gocli project info --include "**/*.go" --include "**/*.md"

  # Exclude specified directory/file patterns
  gocli project info --exclude "vendor/**" --exclude "**/testdata/**"

  # Do not respect .gitignore (default is to respect it)
  gocli project info --no-gitignore

  # Explicitly respect .gitignore (default is true, usually not needed)
  gocli project info --gitignore

  # Follow symbolic links
  gocli project info --follow-symlinks

  # Limit the maximum file size (skips files >1MB)
  gocli project info --max-file-size 1048576

  # Specify the number of concurrent workers
  gocli project info --concurrency 8

  # Disable function/struct statistics (for Go)
  gocli project info --with-funcs=false --with-structs=false

  # Include detailed information for each file (more useful with JSON)
  gocli project info --with-files --json
`,
		Run: func(cmd *cobra.Command, args []string) {
			// determine JSON output
			jsonOut, _ := cmd.Flags().GetBool("json")
			if lf, _ := cmd.Flags().GetBool("language-files"); lf { // auto enable JSON
				jsonOut = true
			}
			if cmd.Flags().Changed("lang-specific") { // 用户显式使用
				jsonOut = true
			}

			// normalize include/exclude patterns so they match collector's toRelSlash output
			normalize := func(raw string) string {
				r := strings.TrimSpace(raw)
				if r == "" {
					return ""
				}
				// convert backslashes to forward slashes
				r = strings.ReplaceAll(r, "\\", "/")
				// strip leading ./ or .\
				if after, ok := strings.CutPrefix(r, "./"); ok {
					r = after
				}
				if after, ok := strings.CutPrefix(r, ".\\"); ok {
					r = after
				}
				return r
			}

			if incl, err := cmd.Flags().GetStringSlice("include"); err == nil {
				clean := make([]string, 0, len(incl))
				for _, p := range incl {
					if p2 := normalize(p); p2 != "" {
						clean = append(clean, p2)
					}
				}
				infoOptions.Include = clean
			}
			if excl, err := cmd.Flags().GetStringSlice("exclude"); err == nil {
				clean := make([]string, 0, len(excl))
				for _, p := range excl {
					if p2 := normalize(p); p2 != "" {
						clean = append(clean, p2)
					}
				}
				infoOptions.Exclude = clean
			}

			noGitignore, _ := cmd.Flags().GetBool("no-gitignore")
			infoOptions.RespectGitignore = !noGitignore

			if err := project.ExecuteInfoCommand(gocliCtx, infoOptions, args, jsonOut, !quiet, cmd.OutOrStdout()); err != nil {
				cmd.PrintErrf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	projectAddCmd  = &cobra.Command{Use: "add", Short: "Add a dependency to the Go project"}
	projectTestCmd = &cobra.Command{Use: "test", Short: "Run tests for the Go project"}
	projectLintCmd = &cobra.Command{
		Use:   "lint",
		Short: "Lint the Go project",
		Long: `gocli project lint checks the Go project for common issues and style violations.(use golangci-lint)

Examples:
  gocli project lint

  # Fix issues
  gocli project lint --fix

  # List all available linters
  gocli project lint --list

  # Print used config path
  gocli project lint --config-path

  # Verify configuration against JSON schema
  gocli project lint --verify

  # Specify the configuration file path
  gocli project lint --config ./path/to/config.yaml

`,
		Run: func(cmd *cobra.Command, _ []string) {
			lintOptions.Verbose = gocliCtx.Config.App.Verbose
			err := project.RunLint(lintOptions, cmd.OutOrStdout())
			if err != nil {
				log.Warn().Msg("have some lint issues")
				os.Exit(1)
			}
		},
	}
	projectFmtCmd = &cobra.Command{
		Use:   "fmt",
		Short: "Format the Go project",
		Long:  `gocli project fmt formats the Go project code (use golangci-lint).`,

		Example: `
  gocli project fmt

  # List all available formatters
  gocli project fmt --list

`,
		Run: func(cmd *cobra.Command, args []string) {
			fmtOptions.Verbose = gocliCtx.Config.App.Verbose
			if len(args) > 0 { // 若用户传入路径，取第一个作为路径
				fmtOptions.Path = args[0]
			}
			err := project.RunFmt(fmtOptions, cmd.OutOrStdout())
			if err != nil {
				log.Warn().Msg("have some format issues")
				os.Exit(1)
			}
		},
	}
	projectUpdateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update dependencies of the Go project",
		Long: `Update the dependencies of the Go project (use 'go get -u' under the hood).

Examples:
  # Update default all dependencies
  gocli project update
  gocli project update ./...

  # Update specific module
  gocli project update github.com/charmbracelet/lipgloss
`,
		Run: func(cmd *cobra.Command, args []string) {
			opts := updateOptions
			if gocliCtx.Config.App.Verbose {
				opts.Verbose = true
			}
			if err := project.RunUpdate(opts, cmd.OutOrStdout(), args); err != nil {
				log.Error().Err(err).Msg("failed to run project update")
				os.Exit(1)
			}
		},
	}
	projectDepsCmd = &cobra.Command{
		Use:   "deps",
		Short: "Manage dependencies of the Go project",
		Long: `gocli project deps provides commands to manage the dependencies of a Go project.

Examples:
  gocli project deps
  gocli project deps --tree
  gocli project deps --graph
  gocli project deps --json
  gocli project deps --tidy
  gocli project deps --vendor
  gocli project deps --download
  gocli project deps --verify
  gocli project deps --why ./...
  gocli project deps --why --why-module std
`,
		Aliases: []string{"dep", "mod"},
		Run: func(cmd *cobra.Command, args []string) {
			opts := depsOptions
			if gocliCtx.Config.App.Verbose {
				opts.Verbose = true
			}
			var b strings.Builder
			if err := project.RunDeps(opts, &b, args); err != nil {
				log.Error().Err(err).Msg("failed to run project deps")
				os.Exit(1)
			}
			output := b.String()
			// JSON: pass-through colorize; others: print raw (tree/graph/tidy/verify/why etc.)
			if opts.JSON {
				_ = style.PrintJSONLine(cmd.OutOrStdout(), output)
				return
			}
			trimmed := strings.TrimRight(output, "\n")
			if trimmed != "" {
				cmd.Print(trimmed)
				cmd.Println()
			}
		},
	}
	projectDocCmd = &cobra.Command{
		Use:   "doc [path|import]",
		Short: "Show docs like 'go doc', with extras",
		Long: `Display package or file documentation with enriched rendering and extra conveniences.

gocli project doc provides a superset of 'go doc' and also supports:
- Rendering Markdown files (README, docs/*.md) with selectable output styles (plain, markdown, html).
- Rendering Go package documentation for local packages, relative paths, and remote module paths (e.g. gorm.io/gorm).
- Including test files, examples and benchmark docs via flags so you can inspect *_test.go information.
- Output to a file (via -o) for sharing or further processing, and theming for markdown/html renderers.

Key behaviors:
- If the target is a directory or package import path, package-level docs and symbols are printed.
- If the target is a Markdown file, it will be rendered according to the chosen style/mode.
- --tests (-t) will include *_test.go symbols (this also auto-enables --examples unless explicitly set).
- --examples (-e) focuses on example functions and usage snippets.
- Use --style to pick the renderer (plain for terminal, markdown for markdown output, html for HTML).
- Use --mode to control parsing mode (godoc for Go-style docs, markdown to treat inputs as Markdown).

When to use:
- Quick local inspection: 'gocli project doc ./cmd' to view package docs while developing.
- Readme preview: 'gocli project doc README.md --style=markdown' to validate generated markdown.
- Third-party lookup: 'gocli project doc gorm.io/gorm' to fetch and show documentation for remote modules.
- Produce shareable HTML: 'gocli project doc ./pkg -o docs/pkg.html --style=html'.

Examples:
  # Show docs for the current module
  gocli project doc .
  gocli project doc ./cmd
  gocli project doc main.go

  # Show docs for a specific package or import path
  gocli project doc fmt
  gocli project doc go/doc
  gocli project doc gorm.io/gorm

  # Render a markdown file (mode will auto set to markdown when extension is .md or .markdown)
  gocli project doc ./README.md --style=markdown -o README_rendered.md

  # Include tests and examples
  gocli project doc ./cmd --tests
  gocli project doc ./cmd --examples

Notes:
- For remote package docs the tool may need network access to fetch module source (behaves like 'go list'/'go doc').
- Large outputs can be redirected to a file using -o. Themes and --width can help produce readable markdown/HTML.
`,
		Run: func(cmd *cobra.Command, args []string) {
			gocliCtx.Config.Doc = docOptions
			if len(args) == 0 {
				_ = cmd.Help()
				os.Exit(0)
			}

			if err := project.RunDoc(gocliCtx, docOptions, cmd.OutOrStdout(), args); err != nil {
				log.Error().Err(err).Msg("failed to run project doc")
				os.Exit(1)
			}
		},
	}
)

func addInitFlags(cmd *cobra.Command, opts *project.InitOptions) {
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List available templates")
	cmd.Flags().StringVarP(&opts.Format, "format", "f", "yaml", "Output format (json|yaml)")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output in JSON format")
	cmd.Flags().BoolVarP(&opts.YAML, "yaml", "y", false, "Output in YAML format")
}

// addBuildRunFlags adds the shared build and run flags to the given command.
func addBuildRunFlags(cmd *cobra.Command, opts *project.BuildRunOptions) {
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Output file name")
	cmd.Flags().StringVarP(&opts.ChangeDir, "changedir", "C", "", "Change to dir before running the command")
	cmd.Flags().BoolVarP(&opts.A, "force-rebuild", "a", false, "Force rebuilding of packages that are already up-to-date")
	cmd.Flags().BoolVarP(&opts.N, "dry-run", "n", false, "Print the commands but do not run them")
	cmd.Flags().IntVarP(&opts.P, "parallel", "p", 0, "Number of programs to run in parallel (default: GOMAXPROCS)")
	cmd.Flags().BoolVar(&opts.Race, "race", false, "Enable data race detection")
	cmd.Flags().BoolVar(&opts.Msan, "msan", false, "Enable memory sanitizer")
	cmd.Flags().BoolVar(&opts.Asan, "asan", false, "Enable address sanitizer")
	cmd.Flags().BoolVar(&opts.Cover, "cover", false, "Enable code coverage analysis")
	cmd.Flags().StringVar(&opts.Covermode, "covermode", "", "Set coverage analysis mode: set, count, atomic")
	cmd.Flags().StringVar(&opts.Coverpkg, "coverpkg", "", "Apply coverage analysis to each package matching the patterns")
	cmd.Flags().BoolVar(&opts.Work, "work", false, "Print the name of the temporary work directory and do not delete it")
	cmd.Flags().BoolVarP(&opts.X, "print-commands", "x", false, "Print the commands")
	cmd.Flags().StringVar(&opts.Asmflags, "asmflags", "", `Arguments to pass on to the compiler`)
	cmd.Flags().StringVar(&opts.Buildmode, "buildmode", "default", "Build mode to use")
	cmd.Flags().StringVar(&opts.Buildvcs, "buildvcs", "", `Whether to stamp binaries with version control information ("true", "false", "auto")`)
	cmd.Flags().StringVar(&opts.Compiler, "compiler", "", `Name of compiler to use, as in runtime.Compiler`)
	cmd.Flags().StringVar(&opts.Gcflags, "gcflags", "", `Arguments to pass on to the compiler`)
	cmd.Flags().StringVar(&opts.Installsuffix, "installsuffix", "", "A suffix to use in the name of the package installation directory")
	cmd.Flags().StringVar(&opts.Ldflags, "ldflags", "", `Arguments to pass on to the linker`)
	cmd.Flags().BoolVar(&opts.Linkshared, "linkshared", false, "Build code that will be linked against shared libraries")
	cmd.Flags().StringVar(&opts.Mod, "mod", "", `Module download mode to use: "readonly", "vendor", or "mod"`)
	cmd.Flags().StringVar(&opts.Tags, "tags", "", "A comma-separated list of build tags to consider satisfied")
	cmd.Flags().BoolVar(&opts.Trimpath, "trimpath", false, "Remove all file system paths from the resulting executable")

	// --- Built-in templates ---
	cmd.Flags().BoolVar(&opts.ReleaseBuild, "release-mode", false, "Build in release mode (remove debug info)")
	cmd.Flags().BoolVar(&opts.DebugBuild, "debug-mode", false, "Build in debug mode (disable optimizations and enable debug info)")
	cmd.Flags().BoolVarP(&opts.HotReload, "hot-reload", "r", false, "Enable hot reloading of code changes")
	cmd.Flags().BoolVar(&opts.NoGitIgnore, "no-gitignore", false, "Disable .gitignore file filtering during hot reload")
}

func addInfoFlags(cmd *cobra.Command, opts *project.InfoOptions) {
	cmd.Flags().StringSliceVar(&opts.Include, "include", nil, "Only include paths matching these glob patterns (comma or repeated)")
	cmd.Flags().StringSliceVar(&opts.Exclude, "exclude", nil, "Exclude paths matching these glob patterns")
	cmd.Flags().BoolVar(&opts.RespectGitignore, "gitignore", true, "Respect .gitignore rules (disable with --no-gitignore)")
	cmd.Flags().Bool("no-gitignore", false, "Do not respect .gitignore (overrides --gitignore)")
	cmd.Flags().BoolVar(&opts.FollowSymlinks, "follow-symlinks", false, "Follow symbolic links")
	cmd.Flags().Int64Var(&opts.MaxFileSizeBytes, "max-file-size", 0, "Skip files larger than this size in bytes (0 means no limit)")
	cmd.Flags().IntVar(&opts.Concurrency, "concurrency", 0, "Number of concurrent workers (0 uses CPU cores)")
	cmd.Flags().BoolVar(&opts.WithFunctions, "funcs", true, "Count functions for supported languages (Go)")
	cmd.Flags().BoolVar(&opts.WithStructs, "structs", true, "Count structs/types for supported languages (Go)")
	cmd.Flags().BoolVar(&opts.WithFileDetails, "files", false, "Include per-file details in JSON output")

	cmd.Flags().Bool("json", false, "Output result in JSON format (auto-enabled if --language-files or explicit --lang-specific used)")
	cmd.Flags().BoolVar(&opts.WithLanguageDetails, "language-files", false, "Include per-file list inside each language (auto enables --json)")
	cmd.Flags().BoolVar(&opts.WithLanguageSpecific, "lang-specific", true, "Include language specific metadata (e.g. Go imports) (explicit use auto enables --json)")

}

func init() {
	rootCmd.AddCommand(projectCmd)

	addInitFlags(projectInitCmd, &initOptions)

	// Add the shared flags to both build and run commands
	addBuildRunFlags(projectBuildCmd, &buildOptions)
	addBuildRunFlags(projectRunCmd, &runOptions)

	addInfoFlags(projectInfoCmd, &infoOptions)

	// lint flags
	projectListCmd.Flags().BoolVarP(&listOptions.JSON, "json", "j", false, "Output packages as JSON array")
	projectListCmd.Flags().BoolVar(&listOptions.Test, "test", false, "Include test packages (adds -test)")

	// lint flags
	projectLintCmd.Flags().BoolVarP(&lintOptions.List, "list", "l", false, "List all available linters")
	projectLintCmd.Flags().BoolVarP(&lintOptions.Fix, "fix", "f", false, "Fix issues where possible")
	projectLintCmd.Flags().BoolVarP(&lintOptions.Verbose, "verbose", "v", false, "Verbose output (line by line)")
	projectLintCmd.Flags().BoolVarP(&lintOptions.Config.Validate, "verify", "V", false, "Verify configuration against JSON schema")
	projectLintCmd.Flags().BoolVarP(&lintOptions.Config.Path, "config-path", "C", false, "Specify the configuration file path")
	projectLintCmd.Flags().StringVarP(&lintOptions.ConfigPath, "config", "c", "", "Specify the configuration file path")

	// fmt flags
	projectFmtCmd.Flags().BoolVarP(&fmtOptions.List, "list", "l", false, "List all available formatters")
	projectFmtCmd.Flags().StringVarP(&fmtOptions.Path, "path", "p", "", "Target path to format (default current directory)")
	projectFmtCmd.Flags().BoolVarP(&fmtOptions.Verbose, "verbose", "v", false, "Verbose output (line by line)")
	projectFmtCmd.Flags().StringVarP(&fmtOptions.ConfigPath, "config", "c", "", "Specify the configuration file path")

	// update flags
	// Usage: gocli project update --verbose ./...
	projectUpdateCmd.Flags().BoolVarP(&updateOptions.Verbose, "verbose", "v", false, "Verbose output (line by line)")

	// deps flags
	projectDepsCmd.Flags().BoolVarP(&depsOptions.JSON, "json", "j", false, "Output dependencies as JSON (go list -m -json)")
	projectDepsCmd.Flags().BoolVarP(&depsOptions.Update, "update", "u", false, "Check for available updates (adds -u)")
	projectDepsCmd.Flags().BoolVarP(&depsOptions.Tree, "tree", "t", false, "Display dependency tree (from 'go mod graph')")
	projectDepsCmd.Flags().BoolVarP(&depsOptions.Graph, "graph", "g", false, "Display dependency graph (raw 'go mod graph')")
	projectDepsCmd.Flags().BoolVarP(&depsOptions.Verbose, "verbose", "v", false, "Verbose output")
	projectDepsCmd.Flags().BoolVar(&depsOptions.Tidy, "tidy", false, "Run 'go mod tidy'")
	projectDepsCmd.Flags().BoolVar(&depsOptions.Vendor, "vendor", false, "Run 'go mod vendor'")
	projectDepsCmd.Flags().BoolVar(&depsOptions.Download, "download", false, "Run 'go mod download'")
	projectDepsCmd.Flags().BoolVar(&depsOptions.Verify, "verify", false, "Run 'go mod verify'")
	projectDepsCmd.Flags().BoolVar(&depsOptions.Why, "why", false, "Run 'go mod why' for given targets (defaults to ./... if none)")
	projectDepsCmd.Flags().BoolVar(&depsOptions.WhyModule, "why-module", false, "Explain why modules are needed (adds -m)")
	projectDepsCmd.Flags().BoolVar(&depsOptions.WhyVendor, "why-vendor", false, "Explain use of vendored packages (adds -vendor)")

	// Disable sorting for build and run commands to group flags logically
	projectBuildCmd.Flags().SortFlags = false
	projectRunCmd.Flags().SortFlags = false

	// doc flags - bind directly to docOptions where possible to match other commands
	projectDocCmd.Flags().StringVarP((*string)(&docOptions.Style), "style", "s", string(doc.StylePlain), "Render style: plain|markdown|html")
	projectDocCmd.Flags().StringVarP((*string)(&docOptions.Mode), "mode", "m", string(doc.ModeGodoc), "Doc mode: godoc|markdown")
	projectDocCmd.Flags().StringVarP(&docOptions.Output, "output", "o", "", "Output file path (default stdout)")
	projectDocCmd.Flags().BoolVarP(&docOptions.IncludePrivate, "private", "p", false, "Include unexported (private) symbols in analysis")
	projectDocCmd.Flags().BoolVarP(&docOptions.IncludeTests, "tests", "t", false, "Include *_test.go files (auto enables --examples if not set)")
	projectDocCmd.Flags().BoolVarP(&docOptions.IncludeExamples, "examples", "e", false, "Include example functions (auto-enabled by --tests)")
	projectDocCmd.Flags().BoolVar(&docOptions.TOC, "toc", true, "Generate table of contents where applicable")
	projectDocCmd.Flags().StringVar(&docOptions.Theme, "theme", "", "Theme for styled output (markdown renderer)")
	projectDocCmd.Flags().IntVarP(&docOptions.Width, "width", "w", 0, "Render width (0 auto)")
	projectDocCmd.Flags().BoolVarP(&docOptions.Detailed, "detailed", "d", false, "Enable detailed output")

	projectCmd.AddCommand(
		projectInitCmd,
		projectBuildCmd,
		projectRunCmd,
		projectListCmd,
		projectInfoCmd,
		projectAddCmd,
		projectTestCmd,
		projectLintCmd,
		projectFmtCmd,
		projectUpdateCmd,
		projectDepsCmd,
		projectDocCmd,
	)
}
