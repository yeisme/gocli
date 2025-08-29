package project

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/yeisme/gocli/pkg/utils/executor"
)

// AddOptions defines options for adding dependencies.
// It uses `cli` struct tags to automatically generate command-line arguments.
type AddOptions struct {
	// --- Standard `go get` flags ---
	T      bool     `cli:"-t"`       // -t: consider modules needed to build tests
	U      bool     `cli:"-u"`       // -u: update modules providing dependencies
	UPatch bool     `cli:"-u=patch"` // -u=patch: update to patch releases
	Tool   bool     `cli:"-tool"`    // -tool: add tool line to go.mod
	X      bool     `cli:"-x"`       // -x: print commands as they are executed
	Args   []string `cli:"-"`        // Additional arguments to pass to go get

	Verbose bool // Verbose output for gocli itself
}

// buildAddArgsFromOptions dynamically generates command-line arguments from the options struct using reflection.
func buildAddArgsFromOptions(options AddOptions) []string {
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
				// Special handling for UPatch
				if tag == "-u=patch" {
					// If UPatch is true, we need to add -u=patch instead of just -u
					args = append(args, "-u=patch")
				} else {
					args = append(args, tag)
				}
			}
		}
	}

	log.Debug().Msgf("Generated add command-line arguments: %v", args)

	return args
}

// RunAdd executes the add command
func RunAdd(options AddOptions, args []string, out io.Writer) error {
	goArgs := []string{"get"}

	// Add flags from options
	goArgs = append(goArgs, buildAddArgsFromOptions(options)...)

	// Add package arguments
	if len(args) > 0 {
		goArgs = append(goArgs, args...)
	}

	// Add additional arguments
	if len(options.Args) > 0 {
		goArgs = append(goArgs, options.Args...)
	}

	executor := executor.NewExecutor("go", goArgs...)

	if options.Verbose {
		fullCmdString := "go " + strings.Join(goArgs, " ")
		if out != nil {
			fmt.Fprint(out, fullCmdString+"\n")
		} else {
			log.Info().Msg(fullCmdString)
		}
	}

	// Execute the get command
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
