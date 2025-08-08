package project

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/yeisme/gocli/pkg/context"
	"github.com/yeisme/gocli/pkg/utils/hotload"

	"github.com/yeisme/gocli/pkg/tools"
	log2 "github.com/yeisme/gocli/pkg/utils/log"
)

var log log2.Logger

func init() {
	log = log2.GetLogger()
}

// BuildRunOptions defines shared options for building and running a project.
// It uses `cli` struct tags to automatically generate command-line arguments.
type BuildRunOptions struct {
	// --- Standard `go build` flags ---
	Output        string `cli:"-o"`             // -o: specifies the output file name
	A             bool   `cli:"-a"`             // -a: forces rebuilding of packages
	N             bool   `cli:"-n"`             // -n: prints the commands but does not run them
	P             int    `cli:"-p"`             // -p: the number of parallel builds
	Race          bool   `cli:"-race"`          // -race: enables data race detection
	Msan          bool   `cli:"-msan"`          // -msan: enables memory sanitizer
	Asan          bool   `cli:"-asan"`          // -asan: enables address sanitizer
	Cover         bool   `cli:"-cover"`         // -cover: enables code coverage analysis
	Covermode     string `cli:"-covermode"`     // -covermode: sets the mode for coverage analysis
	Coverpkg      string `cli:"-coverpkg"`      // -coverpkg: package pattern for coverage analysis
	V             bool   `cli:"-v"`             // -v: prints the names of packages as they are compiled
	Work          bool   `cli:"-work"`          // -work: prints the temporary work directory and does not remove it
	X             bool   `cli:"-x"`             // -x: prints the commands
	Asmflags      string `cli:"-asmflags"`      // -asmflags: arguments to pass on to go tool asm
	Buildmode     string `cli:"-buildmode"`     // -buildmode: build mode
	Buildvcs      string `cli:"-buildvcs"`      // -buildvcs: whether to stamp binaries with VCS information
	Compiler      string `cli:"-compiler"`      // -compiler: name of compiler to use
	Gcflags       string `cli:"-gcflags"`       // -gcflags: arguments to pass on to go tool compile
	Installsuffix string `cli:"-installsuffix"` // -installsuffix: a suffix to use in the name of the installation directory
	Ldflags       string `cli:"-ldflags"`       // -ldflags: arguments to pass on to go tool link
	Linkshared    bool   `cli:"-linkshared"`    // -linkshared: link against shared libraries
	Mod           string `cli:"-mod"`           // -mod: module download mode
	Tags          string `cli:"-tags"`          // -tags: build tags
	Trimpath      bool   `cli:"-trimpath"`      // -trimpath: remove all file system paths from the resulting executable
	ChangeDir     string `cli:"-C"`             // -C: change to dir before running the command

	// --- Built-in templates ---
	BuildinOptions
}

// BuildinOptions contains templated build options for internal use.
type BuildinOptions struct {
	ReleaseBuild bool // Release mode: removes debug information to reduce binary size (-ldflags="-s -w")
	DebugBuild   bool // Debug mode: disables optimizations and enables race detection for easier debugging
	HotReload    bool // Hot reload: enables automatic reloading of code changes
	NoGitIgnore  bool // No git ignore: disables .gitignore file filtering during hot reload
}

// applyBuildTemplates modifies build options based on built-in templates (Release/Debug).
func applyBuildTemplates(opts *BuildRunOptions) {
	// If both Release and Debug are set, Release takes precedence, and a warning is issued.
	if opts.ReleaseBuild && opts.DebugBuild {
		log.Warn().Msg("Both Release and Debug modes are enabled. Prioritizing Release mode.")
		opts.DebugBuild = false
	}

	if opts.ReleaseBuild {
		log.Info().Msg("Applying Release mode optimizations...")

		// Append release flags instead of overwriting to preserve user-defined flags.
		releaseFlags := "-s -w"
		if opts.Ldflags != "" {
			opts.Ldflags = fmt.Sprintf("%s %s", opts.Ldflags, releaseFlags)
		} else {
			opts.Ldflags = releaseFlags
		}

		// Trimpath is enabled by default in Release mode.
		opts.Trimpath = true

		// Set build mode to default for smaller binaries
		if opts.Buildmode == "" {
			opts.Buildmode = "default"
		}

		log.Debug().Msg("Release mode enabled: applying '-ldflags=\"-s -w\"' and '-trimpath'.")
	}

	if opts.DebugBuild {
		log.Info().Msg("Applying Debug mode configurations...")

		// Add flags to Gcflags to disable optimization and enable debug symbols.
		debugGcflags := "all=-N -l"
		if opts.Gcflags != "" {
			opts.Gcflags = fmt.Sprintf("%s %s", opts.Gcflags, debugGcflags)
		} else {
			opts.Gcflags = debugGcflags
		}

		// Enable Race detection by default.
		opts.Race = true

		// Enable verbose output in debug mode
		opts.V = true

		// Enable work directory preservation for debugging
		opts.Work = true

		// Enable command printing for transparency
		opts.X = true

		log.Debug().Msg("Debug mode enabled: applying '-gcflags=\"all=-N -l\"', '-race', '-v', '-work', and '-x'.")
	}
}

// buildArgsFromOptions dynamically generates command-line arguments from the options struct using reflection.
func buildArgsFromOptions(options BuildRunOptions) []string {
	var args []string
	val := reflect.ValueOf(options)
	typ := val.Type()

	// First, apply built-in templates to modify values in the options struct.
	applyBuildTemplates(&options)

	// Re-get the ValueOf, as applyBuildTemplates may have modified the options.
	val = reflect.ValueOf(options)

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		tag := fieldType.Tag.Get("cli")

		// Skip if there is no cli tag or the tag is "-".
		if tag == "" || tag == "-" {
			continue
		}

		// Handle based on field type.
		switch field.Kind() {
		case reflect.Bool:
			if field.Bool() {
				args = append(args, tag)
			}
		case reflect.String:
			if !field.IsZero() {
				// Special handling for buildmode, only add it if it's not "default".
				if fieldType.Name == "Buildmode" && field.String() == "default" {
					continue
				}
				args = append(args, tag, field.String())
			}
		case reflect.Int:
			if !field.IsZero() {
				args = append(args, tag, strconv.Itoa(int(field.Int())))
			}
		}
	}

	log.Debug().Msgf("Generated command-line arguments: %v", args)

	return args
}

// runGoCommand runs a go command using tools.Executor. (This function remains unchanged)
func runGoCommand(options BuildRunOptions, goCmdArgs []string) error {
	executor := tools.NewExecutor("go", goCmdArgs...)
	if options.ChangeDir != "" {
		executor.WithDir(options.ChangeDir)
	}

	if options.N || options.X {
		fullCmdString := "go " + strings.Join(goCmdArgs, " ")
		if options.ChangeDir != "" {
			log.Info().Str("dir", options.ChangeDir).Msg(fullCmdString)
		} else {
			log.Info().Msg(fullCmdString)
		}
	}

	if options.N {
		return nil
	}

	stdout, stderr, err := executor.Run()
	if stdout != "" {
		for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
			log.Info().Msg(line)
		}
	}

	if err == nil && stderr != "" {
		for _, line := range strings.Split(strings.TrimSpace(stderr), "\n") {
			log.Warn().Msg(line)
		}
	}
	return err
}

// executeGoProcessCommand generalizes the execution of "go build" and "go run" commands. (This function remains unchanged)
func executeGoProcessCommand(command string, options BuildRunOptions, args []string) error {
	cmdArgs := []string{command}
	cmdArgs = append(cmdArgs, buildArgsFromOptions(options)...)

	if len(args) > 0 {
		cmdArgs = append(cmdArgs, args...)
	} else {
		cmdArgs = append(cmdArgs, ".")
	}

	return runGoCommand(options, cmdArgs)
}

// 热重启循环，监听变更并自动执行 build/run
func hotReloadLoop(gocliCtx *context.GocliContext, options BuildRunOptions, runFunc func() error) error {
	hotloadConfig := gocliCtx.Config.App.Hotload

	// 如果指定了 --no-gitignore 参数，则覆盖配置中的 git_ignore 设置
	if options.NoGitIgnore {
		hotloadConfig.GitIgnore = false
		log.Info().Msg("[HotReload] --no-gitignore flag specified, disabling .gitignore filtering")
	}

	// 检查热加载是否启用
	if !hotloadConfig.Enabled {
		log.Warn().Msg("[HotReload] Hot reload is disabled in configuration")
		return runFunc() // 直接执行一次，不进行热加载
	}

	// 执行初始构建/运行
	log.Info().Msg("[HotReload] Executing initial build/run...")
	if err := runFunc(); err != nil {
		log.Error().Msgf("[HotReload] Initial execution failed: %v", err)
		return err
	}

	// 使用配置中的监听目录，默认为当前目录
	watchDir := hotloadConfig.Dir
	if watchDir == "" {
		var err error
		watchDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	log.Info().Msgf("[HotReload] Start watching: %s (recursive=%t, git_ignore=%t)",
		watchDir, hotloadConfig.Recursive, hotloadConfig.GitIgnore)
	log.Debug().Msgf("[HotReload] Configuration - Filter: %v, IgnorePatterns: %v, Debounce: %dms",
		hotloadConfig.Filter, hotloadConfig.IgnorePatterns, hotloadConfig.Debounce)

	// 使用配置化的热加载监听器
	return hotload.WatchWithConfig(hotloadConfig, func() {
		log.Info().Msg("[HotReload] Change detected, restarting...")
		if err := runFunc(); err != nil {
			log.Error().Msgf("[HotReload] Execution failed: %v", err)
		}
	})
}

// ExecuteBuildCommand uses the new executeGoProcessCommand. (This function remains unchanged)
func ExecuteBuildCommand(gocliCtx *context.GocliContext, options BuildRunOptions, args []string) error {
	if options.HotReload {
		return hotReloadLoop(gocliCtx, options, func() error {
			return executeGoProcessCommand("build", options, args)
		})
	}
	return executeGoProcessCommand("build", options, args)
}

// ExecuteRunCommand uses the new executeGoProcessCommand. (This function remains unchanged)
func ExecuteRunCommand(gocliCtx *context.GocliContext, options BuildRunOptions, args []string) error {
	if options.HotReload {
		return hotReloadLoop(gocliCtx, options, func() error {
			return executeGoProcessCommand("run", options, args)
		})
	}
	return executeGoProcessCommand("run", options, args)
}
