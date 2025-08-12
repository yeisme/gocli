package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/project"
)

var (
	buildOptions project.BuildRunOptions
	runOptions   project.BuildRunOptions
	infoOptions  project.InfoOptions
	fmtOptions   project.FmtOptions
	lintOptions  project.LintOptions

	projectCmd = &cobra.Command{
		Use:     "project",
		Short:   "Manage Go projects",
		Long:    `gocli project allows you to manage your Go projects, including creating, building, and running them.`,
		Aliases: []string{"p", "pj"},
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
		Use:   "list [flags]",
		Short: "List Go projects",
		Example: strings.TrimSpace(`
  gocli project list
  gocli project list --json
  gocli project list --verbose
`),
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
			jsonOut, _ := cmd.Flags().GetBool("json")
			if lf, _ := cmd.Flags().GetBool("language-files"); lf { // auto enable JSON
				jsonOut = true
			}
			if cmd.Flags().Changed("lang-specific") { // 用户显式使用
				jsonOut = true
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

`,
		Run: func(cmd *cobra.Command, _ []string) {
			lintOptions.Verbose = gocliCtx.Config.App.Verbose
			err := project.RunLint(lintOptions, cmd.OutOrStdout())
			if err != nil {
				cmd.PrintErrf("Error: %v\n", err)
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
				cmd.PrintErrf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	projectUpdateCmd = &cobra.Command{Use: "update", Short: "Update dependencies of the Go project"}
	projectDepsCmd   = &cobra.Command{Use: "deps", Short: "Manage dependencies of the Go project"}
	projectDocCmd    = &cobra.Command{Use: "doc", Short: "Generate documentation for the Go project"}
)

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

	// Add the shared flags to both build and run commands
	addBuildRunFlags(projectBuildCmd, &buildOptions)
	addBuildRunFlags(projectRunCmd, &runOptions)

	addInfoFlags(projectInfoCmd, &infoOptions)

	// lint flags
	projectLintCmd.Flags().BoolVar(&lintOptions.List, "list", false, "List all available linters")
	projectLintCmd.Flags().BoolVar(&lintOptions.Fix, "fix", false, "Fix issues where possible")
	projectLintCmd.Flags().BoolVar(&lintOptions.Verbose, "verbose", false, "Verbose output (line by line)")

	// fmt flags
	projectFmtCmd.Flags().BoolVar(&fmtOptions.List, "list", false, "List all available formatters")
	projectFmtCmd.Flags().StringVarP(&fmtOptions.Path, "path", "p", "", "Target path to format (default current directory)")
	projectFmtCmd.Flags().BoolVar(&fmtOptions.Verbose, "verbose", false, "Verbose output (line by line)")

	// Disable sorting for build and run commands to group flags logically
	projectBuildCmd.Flags().SortFlags = false
	projectRunCmd.Flags().SortFlags = false

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
