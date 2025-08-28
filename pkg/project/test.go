package project

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/yeisme/gocli/pkg/tools"
)

// TestOptions defines options for running tests.
// It uses `cli` struct tags to automatically generate command-line arguments.
type TestOptions struct {
	// --- Standard `go test` flags ---
	V            bool     `cli:"-v"`            // -v: verbose output
	Run          string   `cli:"-run"`          // -run: run only tests matching pattern
	Bench        string   `cli:"-bench"`        // -bench: run benchmarks matching pattern
	Count        int      `cli:"-count"`        // -count: run each test count times
	Timeout      string   `cli:"-timeout"`      // -timeout: timeout for test execution
	Short        bool     `cli:"-short"`        // -short: tell long-running tests to shorten their run time
	Failfast     bool     `cli:"-failfast"`     // -failfast: stop on first test failure
	Parallel     int      `cli:"-parallel"`     // -parallel: maximum test parallelism
	Cover        bool     `cli:"-cover"`        // -cover: enable code coverage
	Covermode    string   `cli:"-covermode"`    // -covermode: coverage mode
	Coverpkg     string   `cli:"-coverpkg"`     // -coverpkg: package pattern for coverage
	Coverprofile string   `cli:"-coverprofile"` // -coverprofile: write coverage profile to file
	Race         bool     `cli:"-race"`         // -race: enable race detector
	JSON         bool     `cli:"-json"`         // -json: output in JSON format
	C            bool     `cli:"-c"`            // -c: compile test binary but do not run
	O            string   `cli:"-o"`            // -o: output binary name
	Args         []string `cli:"-"`             // -args: pass remaining arguments to test binary

	// --- Build-related flags ---
	Tags      string `cli:"-tags"` // -tags: build tags
	Mod       string `cli:"-mod"`  // -mod: module download mode
	ChangeDir string `cli:"-C"`    // -C: change to dir before running the command

	Verbose bool // Verbose output for gocli itself
}

// buildTestArgsFromOptions dynamically generates command-line arguments from the options struct using reflection.
func buildTestArgsFromOptions(options TestOptions) []string {
	var args []string

	val := reflect.ValueOf(options)
	typ := val.Type()

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
				args = append(args, tag, field.String())
			}
		case reflect.Int:
			if !field.IsZero() {
				args = append(args, tag, strconv.Itoa(int(field.Int())))
			}
		}
	}

	log.Debug().Msgf("Generated test command-line arguments: %v", args)

	return args
}

// RunTest executes the test command
func RunTest(options TestOptions, args []string, out io.Writer) error {
	goArgs := []string{"test"}
	goArgs = append(goArgs, buildTestArgsFromOptions(options)...)

	// Add package arguments
	if len(args) == 0 {
		goArgs = append(goArgs, "./...")
	} else {
		goArgs = append(goArgs, args...)
	}

	// Handle -args flag: everything after -args goes to the test binary
	if len(options.Args) > 0 {
		goArgs = append(goArgs, "-args")
		goArgs = append(goArgs, options.Args...)
	}

	executor := tools.NewExecutor("go", goArgs...)
	if options.ChangeDir != "" {
		executor.WithDir(options.ChangeDir)
	}

	if options.Verbose {
		fullCmdString := "go " + strings.Join(goArgs, " ")
		if options.ChangeDir != "" {
			log.Info().Str("dir", options.ChangeDir).Msg(fullCmdString)
		} else {
			log.Info().Msg(fullCmdString)
		}
	}

	// Execute the test command
	stdout, stderr, err := executor.Run()

	// Output results
	if stdout != "" {
		if out != nil {
			fmt.Fprint(out, stdout)
		} else {
			for line := range strings.SplitSeq(strings.TrimSpace(stdout), "\n") {
				log.Info().Msg(line)
			}
		}
	}

	if stderr != "" {
		if out != nil {
			fmt.Fprint(out, stderr)
		} else {
			for line := range strings.SplitSeq(strings.TrimSpace(stderr), "\n") {
				log.Warn().Msg(line)
			}
		}
	}

	return err
}
