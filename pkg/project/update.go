package project

import (
	"bufio"
	"io"
	"strings"

	"github.com/yeisme/gocli/pkg/style"
	"github.com/yeisme/gocli/pkg/utils/deps"
)

// UpdateOptions holds the options for updating dependencies.
type UpdateOptions struct {
	Verbose bool
}

// RunUpdate executes the update command with the given options.
func RunUpdate(opts UpdateOptions, out io.Writer, args []string) error {
	// Spinner while updating
	sp := style.NewSpinner(out, "Updating dependencies")
	sp.Start()

	// Respect default behavior from deps.RunGoUpdate: pass nil to mean "./..."
	var runArgs []string
	if len(args) > 0 {
		runArgs = args
	} else {
		runArgs = nil
	}

	// 先执行 go mod tidy
	if _, err := deps.RunGoModTidy(); err != nil {
		return err
	}

	output, err := deps.RunGoUpdate(runArgs)
	// Stop spinner before any further output
	sp.Stop()

	if err != nil {
		// Best-effort styled error heading, then return
		_ = style.PrintHeading(out, "Update Failed")
		// Print stderr if available via logger; keep command return for caller
		if opts.Verbose {
			log.Error().Err(err).Msg("go get -u returned error")
		}
		return err
	}

	// Success heading
	_ = style.PrintHeading(out, "Dependencies Updated")

	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		// Many go versions print little on success; give a friendly note
		log.Info().Msg("No changes reported by go get -u.")
		return nil
	}

	// Pretty print each non-empty line as a bullet list
	lines := make([]string, 0, 16)
	scanner := bufio.NewScanner(strings.NewReader(trimmed))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > 0 {
		// Convert []string -> []any for PrintList
		items := make([]any, len(lines))
		for i := range lines {
			items[i] = lines[i]
		}
		_ = style.PrintList(out, items...)
	}
	return nil
}
