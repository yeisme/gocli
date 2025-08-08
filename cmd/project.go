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

	projectCmd = &cobra.Command{
		Use:     "project",
		Short:   "Manage Go projects",
		Long:    `gocli project allows you to manage your Go projects, including creating, building, and running them.`,
		Aliases: []string{"p", "mod"},
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
		Long:  `gocli project build compiles the Go project. You can specify packages to build, or it defaults to the current directory.`,
		Run: func(cmd *cobra.Command, args []string) {
			buildOptions.V = gocliCtx.Config.App.Verbose
			if err := project.ExecuteBuildCommand(gocliCtx, buildOptions, args); err != nil {
				cmd.PrintErrf("Error: %v\n", err)
				os.Exit(1)
			}
		},
		Example: strings.TrimSpace(`
  gocli project build
  gocli project build -o myapp ./cmd/server
  gocli project build --race
  gocli project build --tags "sqlite,json" --ldflags "-s -w"
  gocli project build --gcflags "-N -l"
`),
	}
	projectRunCmd = &cobra.Command{
		Use:   "run [args...] [packages]",
		Short: "Run the Go project",
		Long:  `gocli project run compiles and runs a main Go program.`,
		Run: func(cmd *cobra.Command, args []string) {
			runOptions.V = gocliCtx.Config.App.Verbose
			if err := project.ExecuteRunCommand(gocliCtx, runOptions, args); err != nil {
				cmd.PrintErrf("Error: %v\n", err)
				os.Exit(1)
			}
		},
		Example: strings.TrimSpace(`
  gocli project run
  gocli project run main.go
  gocli project run --race ./cmd/server
`),
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
		Example: strings.TrimSpace(`
  gocli project info
  gocli project info --json
  gocli project info --verbose
`),
	}
	projectAddCmd  = &cobra.Command{Use: "add", Short: "Add a dependency to the Go project"}
	projectTestCmd = &cobra.Command{Use: "test", Short: "Run tests for the Go project"}
	projectLintCmd = &cobra.Command{
		Use:   "lint",
		Short: "Lint the Go project",
		Example: strings.TrimSpace(`
  gocli project lint
  gocli project lint --fix
`),
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

	// --- Goblin options ---

}

func init() {
	rootCmd.AddCommand(projectCmd)

	// Add the shared flags to both build and run commands
	addBuildRunFlags(projectBuildCmd, &buildOptions)
	addBuildRunFlags(projectRunCmd, &runOptions)

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
		projectUpdateCmd,
		projectDepsCmd,
		projectDocCmd,
	)
}
