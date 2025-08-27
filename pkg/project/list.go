package project

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yeisme/gocli/pkg/utils/list"
)

// ListOptions defines the options for the list command.
type ListOptions struct {
	JSON bool
	Test bool
}

// RunList executes the `go list` command with the provided options and writes the output to the specified writer.
func RunList(opts ListOptions, out io.Writer, args []string) error {
	args = normalizeListArgs(args)

	output, err := list.RunGoList(context.Background(), opts, args)
	if err != nil {
		return err
	}
	fmt.Fprint(out, output)
	return nil
}

// normalizeListArgs ensures the first (and each provided) argument is a valid path / pattern
// understood by `go list`. Behaviour:
//   - No args => ["./..."]
//   - "." => "./..."
//   - "all" / "std" / explicit pattern (contains "..." or wildcard) left unchanged
//   - Directory paths without pattern get "/..." appended so that all sub-packages are included
//   - File paths (ending with .go) are kept as-is (go list can handle them when paired with -f, but we pass through)
//   - Empty strings are ignored
func normalizeListArgs(in []string) []string {
	if len(in) == 0 {
		return []string{"./..."}
	}
	out := make([]string, 0, len(in))
	for _, a := range in {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		switch a {
		case ".":
			out = append(out, "./...")
			continue
		case "all", "std":
			out = append(out, a)
			continue
		}
		// keep patterns containing ... or wildcards
		if strings.Contains(a, "...") || strings.ContainsAny(a, "*?[") {
			out = append(out, a)
			continue
		}
		// If it's an absolute or relative directory path, append /... to include subpackages
		info, err := os.Stat(a)
		if err == nil && info.IsDir() {
			// normalize to relative form for nicer output if within module
			if abs, err2 := filepath.Abs(a); err2 == nil {
				if wd, err3 := os.Getwd(); err3 == nil {
					if rel, err4 := filepath.Rel(wd, abs); err4 == nil {
						a = rel
					}
				}
			}
			// Avoid duplicating trailing separator
			a = filepath.ToSlash(strings.TrimSuffix(a, "/")) + "/..."
			out = append(out, a)
			continue
		}
		// Else leave as-is (could be module path or specific package)
		out = append(out, a)
	}
	if len(out) == 0 {
		return []string{"./..."}
	}
	return out
}
