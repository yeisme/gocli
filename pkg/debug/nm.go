package debug

import (
	"fmt"
	"io"
	"strings"

	"github.com/yeisme/gocli/pkg/utils/executor"
)

// NMOptions holds flags mapping for `go tool nm`.
type NMOptions struct {
	Numeric bool   // -n (alias for -sort address)
	Size    bool   // -size
	Sort    string // -sort {address,name,none,size}
	Type    bool   // -type
	Verbose bool   // internal verbose
}

// RunNM executes `go tool nm` with provided options and files.
// Output is streamed to stdout/stderr writers.
func RunNM(stderr io.Writer, stdout io.Writer, opt NMOptions, files ...string) error {
	args := []string{"tool", "nm"}

	if opt.Numeric {
		args = append(args, "-n")
	}
	if opt.Size {
		args = append(args, "-size")
	}
	if s := strings.TrimSpace(opt.Sort); s != "" && s != "name" { // 'name' is default, pass others explicitly
		args = append(args, "-sort", s)
	}
	if opt.Type {
		args = append(args, "-type")
	}

	args = append(args, files...)

	if opt.Verbose {
		fmt.Fprintf(stderr, "running: go %s\n", strings.Join(args, " "))
	}

	return executor.NewExecutor("go", args...).RunStreaming(stdout, stderr)
}
