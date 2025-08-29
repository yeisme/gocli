package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeisme/gocli/pkg/project"
	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/doc"
	"github.com/yeisme/gocli/pkg/utils/executor"
)

var (
	initOptions   project.InitOptions
	buildOptions  project.BuildRunOptions
	runOptions    project.BuildRunOptions
	listOptions   project.ListOptions
	infoOptions   project.InfoOptions
	lintOptions   project.LintOptions
	fmtOptions    project.FmtOptions
	updateOptions project.UpdateOptions
	depsOptions   project.DepsOptions
	docOptions    project.DocOptions
	addOptions    project.AddOptions
	testOptions   project.TestOptions

	projectCmd = &cobra.Command{
		Use:     "project",
		Short:   "Manage Go projects",
		Long:    `gocli project allows you to manage your Go projects, including creating, building, and running them.`,
		Aliases: []string{"p", "pj", "go"},
	}

	projectInitCmd = &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize a new Go project",
		Long: `
Initialize a new Go project with the necessary files and directories.

Basic usage:
  gocli project init [name] [flags]

Examples:
  # 1. Initialize in current directory (module name inferred from path)
  gocli project init .

  # 2. Initialize a new project directory (creates folder if not exists)
  gocli project init myapp

  # 3. Specify target directory explicitly
  gocli project init --dir ./services/api myapi

  # 4. List available templates
  gocli project init --list

  # 5. Use a template (copies template files then runs go mod init)
  gocli project init myweb --template basic

  # 6. Disable git init (default is enabled when --git provided / true)
  gocli project init myapp --git

  # 7. Enable extra tool configs
  gocli project init myapp --go-task --gocli --goreleaser --docker --makefile

  # 8. Add license & author meta
  gocli project init mylib --license MIT --author "Alice" --email alice@example.com

  # 9. Output templates as JSON / YAML when listing
  gocli project init --list --json
  gocli project init --list --format yaml

  # 10. Force overwrite existing files from template
  gocli project init myapp --template basic --force

  # 11. Initialize using a template name passed as --type when no explicit --template
  #     (If the value matches a registered template and type not go, it's treated as template)
  gocli project init basic --type basic

  # 12. Combine: create dir, apply template, init task & goreleaser
  gocli project init svc-user --dir ./services/user --template api --go-task --goreleaser

Notes:
  - If go.mod already exists in the target directory, go mod init is skipped.
  - --force overwrites files that already exist when copying template content.
  - --json / --yaml only affect template list output (when --list specified).
  - Author/email/license insertion depends on template support.
`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := project.ExecuteInitCommand(gocliCtx, args, initOptions, cmd.OutOrStdout()); err != nil {
				// 如果是 ExecError（包含 stderr），直接把格式化后的错误作为消息打印，避免 zerolog 将换行转义
				if ee, ok := err.(*executor.ExecError); ok {
					log.Error().Msgf("failed to initialize project: %s", ee.Error())
				} else {
					log.Error().Err(err).Msg("failed to initialize project")
				}
			}
		},
		Args: cobra.MaximumNArgs(1),
	}
	projectBuildCmd = &cobra.Command{
		Use:   "build [args...] [packages]",
		Short: "Build the Go project",
		Long: `
gocli project build compiles Go packages (a superset wrapper of 'go build').

Basic usage:
  gocli project build [flags] [packages]
	When no package/path is specified, the current directory is used. You may pass a
	directory, file, or pattern (e.g. ./..., ./cmd/server).

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

Notes:
  - Most flags map directly to 'go build' counterparts (asmflags/gcflags/ldflags...).
  - --release-mode / --debug-mode are opinionated presets combining common flags.
  - Can be combined with --hot-reload (more commonly used under 'run').
`,
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
		Long: `
gocli project run builds then runs one (or multiple) main entrypoints (main package / main.go).

Basic usage:
  gocli project run [flags] [packages]
	By default runs the main package in the current directory; you can specify files
	or directories (e.g. main.go, ./cmd/server). Flags control build/run behavior and hot reload.

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

Notes:
  - Hot reload is for local dev; for production prefer a static build + external supervisor.
  - --release-mode may also be used here to emulate production flags for a quick run.
  - Use -n / --dry-run to only print the underlying commands.
`,
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
		Long: `
gocli project list lists Go packages under the current module.

Basic usage:
  gocli project list [flags] [patterns]
	By default it expands to './...' to list all packages. Patterns can be provided
	(e.g. ./cmd/..., ./pkg/utils, ./... ).

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
		Long: `
gocli project info analyzes a Go module or directory and prints useful statistics and metadata.

Basic usage:
  gocli project info [flags] [path]
  Use flags to control inclusion/exclusion, output format (JSON), and traversal behavior.

Examples:
  # Analyze current module (human-readable summary)
  gocli project info

  # Analyze a directory and get machine-friendly JSON output
  gocli project info ./ --json

  # Only include Go sources and README files
  gocli project info --include "**/*.go" --include "**/README.md"

  # Exclude vendor and testdata directories
  gocli project info --exclude "vendor/**" --exclude "**/testdata/**"

  # Ignore .gitignore rules (scan everything)
  gocli project info --no-gitignore

  # Follow symbolic links when collecting files
  gocli project info --follow-symlinks

  # Skip very large files (>1MB)
  gocli project info --max-file-size 1048576

  # Increase parallel workers for faster analysis on large repos
  gocli project info --concurrency 8

  # Disable counting functions or structs (reduce work)
  gocli project info --funcs=false --structs=false

  # Include per-file breakdown (useful with --json)
  gocli project info --with-files --json

  # Short-form (shorthand) flags example - equivalent to above long-form examples
  #  - include paths (-i), exclude (-e), respect gitignore (-g), follow symlinks (-L)
  #  - set max file size (-m), concurrency (-C), disable funcs/structs (-F -S)
  #  - include per-file details (-f) and output JSON (-j)
  gocli project info -i "./pkg" -e "vendor/**" -g -L -m 1048576 -C 8 -F=false -S=false -f -j

  # Short-form: include per-language file lists and enable JSON
  gocli project info -i "**/*.go" -l -j

Notes:
  - When using --with-files or explicitly supplying language-specific flags, JSON output is auto-enabled to ensure structured data.
  - Use glob-style patterns for --include/--exclude; Windows backslashes are accepted but forward slashes are recommended.
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
	projectAddCmd = &cobra.Command{
		Use:     "add",
		Short:   "Add a dependency to the Go project",
		Aliases: []string{"get", "g", "a"},
		Run: func(cmd *cobra.Command, args []string) {
			addOptions.Verbose = gocliCtx.Config.App.Verbose
			if err := project.RunAdd(addOptions, args, cmd.OutOrStdout()); err != nil {
				cmd.PrintErrf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	projectTestCmd = &cobra.Command{
		Use:   "test [flags] [packages]",
		Short: "Run tests for the Go project",
		Long: `
gocli project test runs tests for Go packages (wrapper around 'go test').

Basic usage:
  gocli project test [flags] [packages]
	By default runs tests for all packages in the current module. You can specify
	packages to test specific directories or patterns (e.g. ./pkg/..., ./cmd).

Examples:
  # Run all tests in the current module
  gocli project test

  # Run tests for specific packages
  gocli project test ./pkg/utils
  gocli project test ./...

  # Run tests with verbose output
  gocli project test -v

  # Run only tests matching a pattern
  gocli project test -run TestAdd

  # Run benchmarks
  gocli project test -bench .

  # Run tests with race detection
  gocli project test -race

  # Run tests with coverage
  gocli project test -cover
  gocli project test -coverprofile=coverage.out

  # Run tests in parallel with timeout
  gocli project test -parallel 4 -timeout 30s

  # Run tests multiple times
  gocli project test -count 3

  # Run short tests only
  gocli project test -short

  # Stop on first failure
  gocli project test -failfast

  # Output in JSON format
  gocli project test -json

  # Compile test binary without running
  gocli project test -c -o mytest

Notes:
  - Most flags map directly to 'go test' counterparts.
  - Test output follows 'go test' behavior: successful tests show summary only,
    failed tests show detailed output.
  - Supports all standard 'go test' flags for comprehensive test control.
`,
		Run: func(cmd *cobra.Command, args []string) {
			testOptions.Verbose = gocliCtx.Config.App.Verbose
			if err := project.RunTest(testOptions, args, cmd.OutOrStdout()); err != nil {
				cmd.PrintErrf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	projectLintCmd = &cobra.Command{
		Use:   "lint",
		Short: "Lint the Go project",
		Long: `
gocli project lint checks the Go project for common issues and style violations.(use golangci-lint)

Basic usage:
  gocli project lint [flags] [path]

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
		Long: `
gocli project fmt formats the Go project code (use golangci-lint).

Basic usage:
  gocli project fmt [flags] [path]

Examples:
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
		Long: `
Update the dependencies of the Go project (use 'go get -u' under the hood).

Basic usage:
  gocli project update [flags] [packages]

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
		Long: `
gocli project deps provides commands to manage the dependencies of a Go project and perform module maintenance.

Basic usage:
  gocli project deps [flags] [targets]
	Use with no targets to operate on the current module (defaults to ./...).

Examples:

  # 1. List current module dependencies (human readable)
  gocli project deps

  # 2. Output dependencies as JSON (useful for tooling)
  gocli project deps --json
  gocli project deps -j

  # 3. Show dependency tree (visual hierarchical view)
  gocli project deps --tree
  gocli project deps -t

  # 4. Show raw dependency graph (output of 'go mod graph')
  gocli project deps --graph
  gocli project deps -g

  # 5. Run maintenance actions (these modify files): tidy, vendor, download
  gocli project deps --tidy
  gocli project deps -d    # shorthand for tidy
  gocli project deps --vendor
  gocli project deps -n    # shorthand for vendor
  gocli project deps --download
  gocli project deps -w    # shorthand for download

  # 6. Verify module checksums (go mod verify)
  gocli project deps --verify
  gocli project deps -f    # shorthand for verify

  # 7. Check for available updates (adds -u) and output JSON
  gocli project deps --update
  gocli project deps -u -j

  # 8. Explain why packages/modules are required (defaults to ./... when no target provided)
  # - basic why for current module
  gocli project deps --why
  gocli project deps -y

  # 9. Explain why with module-level context (-m / --why-module) and with vendor-aware hints (-V / --why-vendor)
  gocli project deps --why --why-module std
  gocli project deps -y -m std
  gocli project deps --why --why-vendor ./...
  gocli project deps -y -V ./...

  # 10. Verbose output combined with tree/graph views
  gocli project deps --tree --verbose
  gocli project deps -t -v

  # 11. Full example that exercises most flags and uses short forms where available
  # (lists updates in JSON, shows tree, enables verbose output, and runs verify)
  gocli project deps -u -j -t -v -f ./...

Notes:
  - Short flags: -j (json), -u (update), -t (tree), -g (graph), -v (verbose),
	-d (tidy), -n (vendor), -w (download), -f (verify), -y (why), -m (why-module), -V (why-vendor).
  - Maintenance actions like --tidy, --vendor and --download modify module files; run intentionally and commit changes if desired.
  - --why accepts package patterns (e.g. ./... or a specific import path). When no target is provided it defaults to ./...
  - Use --verbose (-v) to get more diagnostic output when combining views (tree/graph/why).
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
		Long: `
Display package or file documentation with enriched rendering and extra conveniences.

Basic usage:
  gocli project doc [flags] <path|import>
	Renders package or file documentation with selectable styles/modes and optional output file.

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
	// List Flags (also output format)
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List available templates")
	cmd.Flags().StringVarP(&opts.Format, "format", "f", "", "Output format (json|yaml|plain|table) only used with --list")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output in JSON format")
	cmd.Flags().BoolVarP(&opts.YAML, "yaml", "y", false, "Output in YAML format")
	cmd.Flags().BoolVarP(&opts.Plain, "plain", "p", true, "Output plain list")
	cmd.Flags().BoolVarP(&opts.Table, "table", "T", false, "Output in table format")

	cmd.Flags().StringVarP(&opts.LangType, "type", "t", "go", "Set project type (go|cpp|python|node|rust|java|php|dotnet, only 'go' supported now)")
	cmd.Flags().StringVarP(&opts.Template, "template", "m", "", "Project template name (use --list to see available templates)")
	cmd.Flags().StringVarP(&opts.Project.Dir, "dir", "d", "", "Project directory (defaults to current directory)")
	cmd.Flags().BoolVarP(&opts.Force, "force", "F", false, "Force overwrite existing files")

	// Project Init
	cmd.Flags().BoolVar(&opts.Project.GoTaskInit, "go-task", false, "Initialize go-task configuration")
	cmd.Flags().BoolVar(&opts.Project.GitInit, "git", false, "Initialize git repository (git init) (use --git=false to disable)")
	cmd.Flags().BoolVar(&opts.Project.GoCLIInit, "gocli", false, "Initialize gocli config (gocli config init)")
	cmd.Flags().BoolVar(&opts.Project.GoreleaserInit, "goreleaser", false, "Initialize Goreleaser config (goreleaser init)")
	cmd.Flags().BoolVar(&opts.Project.DockerInit, "docker", false, "Initialize Docker related files (docker init)")
	cmd.Flags().BoolVar(&opts.Project.MakefileInit, "makefile", false, "Initialize Makefile (makefile init)")
	cmd.Flags().StringVar(&opts.Project.License, "license", "", "License identifier to add (e.g. MIT, Apache-2.0)")
	cmd.Flags().StringVar(&opts.Project.Author, "author", "", "Author name (used in generated files if template supports)")
	cmd.Flags().StringVar(&opts.Project.Email, "email", "", "Author email (used in generated files if template supports)")

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
	// add short aliases for common flags to improve ergonomics
	cmd.Flags().StringSliceVarP(&opts.Include, "include", "i", nil, "Only include paths matching these glob patterns (comma or repeated)")
	cmd.Flags().StringSliceVarP(&opts.Exclude, "exclude", "e", nil, "Exclude paths matching these glob patterns")
	cmd.Flags().BoolVarP(&opts.RespectGitignore, "gitignore", "g", true, "Respect .gitignore rules (disable with --no-gitignore)")
	// keep --no-gitignore without a short alias to avoid confusion with --gitignore
	cmd.Flags().Bool("no-gitignore", false, "Do not respect .gitignore (overrides --gitignore)")
	cmd.Flags().BoolVarP(&opts.FollowSymlinks, "follow-symlinks", "L", false, "Follow symbolic links")
	cmd.Flags().Int64VarP(&opts.MaxFileSizeBytes, "max-file-size", "m", 0, "Skip files larger than this size in bytes (0 means no limit)")
	cmd.Flags().IntVarP(&opts.Concurrency, "concurrency", "C", 0, "Number of concurrent workers (0 uses CPU cores)")
	cmd.Flags().BoolVarP(&opts.WithFunctions, "funcs", "F", true, "Count functions for supported languages (Go)")
	cmd.Flags().BoolVarP(&opts.WithStructs, "structs", "S", true, "Count structs/types for supported languages (Go)")
	cmd.Flags().BoolVarP(&opts.WithFileDetails, "files", "f", false, "Include per-file details in JSON output")

	cmd.Flags().BoolP("json", "j", false, "Output result in JSON format (auto-enabled if --language-files or explicit --lang-specific used)")
	cmd.Flags().BoolVarP(&opts.WithLanguageDetails, "language-files", "l", false, "Include per-file list inside each language (auto enables --json)")
	cmd.Flags().BoolVarP(&opts.WithLanguageSpecific, "lang-specific", "k", true, "Include language specific metadata (e.g. Go imports) (explicit use auto enables --json)")

}

func addDepsFlags(cmd *cobra.Command, opts *project.DepsOptions) {
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output dependencies as JSON (go list -m -json)")
	cmd.Flags().BoolVarP(&opts.Update, "update", "u", false, "Check for available updates (adds -u)")
	cmd.Flags().BoolVarP(&opts.Tree, "tree", "t", false, "Display dependency tree (from 'go mod graph')")
	cmd.Flags().BoolVarP(&opts.Graph, "graph", "g", false, "Display dependency graph (raw 'go mod graph')")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output")
	cmd.Flags().BoolVarP(&opts.Tidy, "tidy", "d", false, "Run 'go mod tidy'")
	cmd.Flags().BoolVarP(&opts.Vendor, "vendor", "n", false, "Run 'go mod vendor'")
	cmd.Flags().BoolVarP(&opts.Download, "download", "w", false, "Run 'go mod download'")
	cmd.Flags().BoolVarP(&opts.Verify, "verify", "f", false, "Run 'go mod verify'")
	cmd.Flags().BoolVarP(&opts.Why, "why", "y", false, "Run 'go mod why' for given targets (defaults to ./... if none)")
	cmd.Flags().BoolVarP(&opts.WhyModule, "why-module", "m", false, "Explain why modules are needed (adds -m)")
	cmd.Flags().BoolVarP(&opts.WhyVendor, "why-vendor", "V", false, "Explain use of vendored packages (adds -vendor)")
}

// addListFlags registers flags for the `project list` command.
func addListFlags(cmd *cobra.Command, opts *project.ListOptions) {
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output packages as JSON array")
	cmd.Flags().BoolVar(&opts.Test, "test", false, "Include test packages (adds -test)")
}

// addAddFlags registers flags for the `project add` command.
func addAddFlags(cmd *cobra.Command, opts *project.AddOptions) {
	cmd.Flags().BoolVarP(&opts.T, "test", "t", false, "Consider modules needed to build tests")
	cmd.Flags().BoolVarP(&opts.U, "update", "u", false, "Update modules providing dependencies")
	cmd.Flags().BoolVar(&opts.UPatch, "update-patch", false, "Update to patch releases (equivalent to -u=patch)")
	cmd.Flags().BoolVar(&opts.Tool, "tool", false, "Add tool line to go.mod")
	cmd.Flags().BoolVarP(&opts.X, "print-commands", "x", false, "Print commands as they are executed")
}

// addTestFlags registers flags for the `project test` command.
func addTestFlags(cmd *cobra.Command, opts *project.TestOptions) {
	cmd.Flags().BoolVarP(&opts.V, "verbose", "v", false, "Verbose output")
	cmd.Flags().StringVarP(&opts.Run, "run", "r", "", "Run only tests matching pattern")
	cmd.Flags().StringVarP(&opts.Bench, "bench", "b", "", "Run benchmarks matching pattern")
	cmd.Flags().IntVarP(&opts.Count, "count", "C", 0, "Run each test count times")
	cmd.Flags().StringVarP(&opts.Timeout, "timeout", "t", "", "Timeout for test execution")
	cmd.Flags().BoolVarP(&opts.Short, "short", "s", false, "Tell long-running tests to shorten their run time")
	cmd.Flags().BoolVarP(&opts.Failfast, "failfast", "f", false, "Stop on first test failure")
	cmd.Flags().IntVarP(&opts.Parallel, "parallel", "p", 0, "Maximum test parallelism")
	cmd.Flags().BoolVar(&opts.Cover, "cover", false, "Enable code coverage")
	cmd.Flags().StringVar(&opts.Covermode, "covermode", "", "Coverage mode: set, count, atomic")
	cmd.Flags().StringVar(&opts.Coverpkg, "coverpkg", "", "Apply coverage to packages matching pattern")
	cmd.Flags().StringVar(&opts.Coverprofile, "coverprofile", "", "Write coverage profile to file")
	cmd.Flags().BoolVar(&opts.Race, "race", false, "Enable race detection")
	cmd.Flags().BoolVarP(&opts.JSON, "json", "j", false, "Output in JSON format")
	cmd.Flags().BoolVar(&opts.C, "compile-only", false, "Compile test binary but do not run")
	cmd.Flags().StringVarP(&opts.O, "output", "o", "", "Output binary name")
	cmd.Flags().StringVarP(&opts.Tags, "tags", "g", "", "Build tags")
	cmd.Flags().StringVarP(&opts.Mod, "mod", "m", "", "Module download mode")
	cmd.Flags().StringVarP(&opts.ChangeDir, "changedir", "D", "", "Change to dir before running")
}

// addLintFlags registers flags for the `project lint` command.
func addLintFlags(cmd *cobra.Command, opts *project.LintOptions) {
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List all available linters")
	cmd.Flags().BoolVarP(&opts.Fix, "fix", "f", false, "Fix issues where possible")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output (line by line)")
	cmd.Flags().BoolVarP(&opts.Config.Validate, "verify", "V", false, "Verify configuration against JSON schema")
	cmd.Flags().BoolVarP(&opts.Config.Path, "config-path", "C", false, "Specify the configuration file path")
	cmd.Flags().StringVarP(&opts.ConfigPath, "config", "c", "", "Specify the configuration file path")
}

// addFmtFlags registers flags for the `project fmt` command.
func addFmtFlags(cmd *cobra.Command, opts *project.FmtOptions) {
	cmd.Flags().BoolVarP(&opts.List, "list", "l", false, "List all available formatters")
	cmd.Flags().StringVarP(&opts.Path, "path", "p", "", "Target path to format (default current directory)")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output (line by line)")
	cmd.Flags().StringVarP(&opts.ConfigPath, "config", "c", "", "Specify the configuration file path")
}

// addUpdateFlags registers flags for the `project update` command.
func addUpdateFlags(cmd *cobra.Command, opts *project.UpdateOptions) {
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output (line by line)")
}

// addDocFlags registers flags for the `project doc` command.
func addDocFlags(cmd *cobra.Command, opts *project.DocOptions) {
	cmd.Flags().StringVarP((*string)(&opts.Style), "style", "s", string(doc.StylePlain), "Render style: plain|markdown|html")
	cmd.Flags().StringVarP((*string)(&opts.Mode), "mode", "m", string(doc.ModeGodoc), "Doc mode: godoc|markdown")
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "", "Output file path (default stdout)")
	cmd.Flags().BoolVarP(&opts.IncludePrivate, "private", "p", false, "Include unexported (private) symbols in analysis")
	cmd.Flags().BoolVarP(&opts.IncludeTests, "tests", "t", false, "Include *_test.go files (auto enables --examples if not set)")
	cmd.Flags().BoolVarP(&opts.IncludeExamples, "examples", "e", false, "Include example functions (auto-enabled by --tests)")
	cmd.Flags().BoolVar(&opts.TOC, "toc", true, "Generate table of contents where applicable")
	cmd.Flags().StringVar(&opts.Theme, "theme", "", "Theme for styled output (markdown renderer)")
	cmd.Flags().IntVarP(&opts.Width, "width", "w", 0, "Render width (0 auto)")
	cmd.Flags().BoolVarP(&opts.Detailed, "detailed", "d", false, "Enable detailed output")
}

// registerProjectFlags centralizes all flag registrations for project subcommands
// and orders them to match the command registration order in init.
func registerProjectFlags() {
	// 1) init
	addInitFlags(projectInitCmd, &initOptions)

	// 2) build
	addBuildRunFlags(projectBuildCmd, &buildOptions)

	// 3) run
	addBuildRunFlags(projectRunCmd, &runOptions)

	// 4) list
	addListFlags(projectListCmd, &listOptions)

	// 5) info
	addInfoFlags(projectInfoCmd, &infoOptions)

	// 6) add (no flags currently)
	addAddFlags(projectAddCmd, &addOptions)

	// 7) test
	addTestFlags(projectTestCmd, &testOptions)

	// 8) lint
	addLintFlags(projectLintCmd, &lintOptions)

	// 9) fmt
	addFmtFlags(projectFmtCmd, &fmtOptions)

	// 10) update
	addUpdateFlags(projectUpdateCmd, &updateOptions)

	// 11) deps
	addDepsFlags(projectDepsCmd, &depsOptions)

	// Keep build/run flag ordering as originally intended
	projectBuildCmd.Flags().SortFlags = false
	projectRunCmd.Flags().SortFlags = false

	addDocFlags(projectDocCmd, &docOptions)
}

func init() {
	rootCmd.AddCommand(projectCmd)

	// register flags and options in the same order as AddCommand below
	registerProjectFlags()

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
