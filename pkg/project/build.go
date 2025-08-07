package project

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

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

// ExecuteBuildCommand uses the new executeGoProcessCommand. (This function remains unchanged)
func ExecuteBuildCommand(options BuildRunOptions, args []string) error {
	return executeGoProcessCommand("build", options, args)
}

// ExecuteRunCommand uses the new executeGoProcessCommand. (This function remains unchanged)
func ExecuteRunCommand(options BuildRunOptions, args []string) error {
	return executeGoProcessCommand("run", options, args)
}
