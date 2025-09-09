package debug

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/yeisme/gocli/pkg/utils/executor"
)

// TraceOptions holds user supplied flags for trace viewing / analysis.
type TraceOptions struct {
	HTTPAddr string // -http
	PProf    string // -pprof
	Debug    string // -d
	Verbose  bool   // internal verbose
}

// RunTrace wraps `go tool trace` with additional validation and output control.
// If pprof type is specified, it prints the generated profile text to stdout.
// Otherwise it launches the trace web UI server (blocking until killed / ctrl+C).
//
// bin may be empty for Go >=1.7 traces.
func RunTrace(stderr io.Writer, stdout io.Writer, opt TraceOptions, bin string, traceFile string) error {
	args := []string{"tool", "trace"}

	if opt.HTTPAddr != "" && opt.PProf == "" { // -http only meaningful when not generating pprof
		args = append(args, "-http", opt.HTTPAddr)
	}
	if opt.PProf != "" { // mutually exclusive with -http per typical usage
		args = append(args, "-pprof", opt.PProf)
	}
	if opt.Debug != "" {
		args = append(args, "-d", opt.Debug)
	}

	if bin != "" {
		args = append(args, bin)
	}
	args = append(args, traceFile)

	if opt.Verbose {
		fmt.Fprintf(stderr, "running: go %s\n", strings.Join(args, " "))
	}

	// When -http is used, go tool trace starts a web server and blocks; we should stream.
	if opt.PProf == "" && opt.HTTPAddr != "" { // interactive server
		// Use exec.Command directly to inherit Ctrl+C (SIGINT) behaviour; still capture stderr.
		cmd := exec.Command("go", args...)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		return cmd.Run()
	}

	// For pprof generation or simple debug printing we can just capture output.
	out, err := executor.NewExecutor("go", args...).CombinedOutput()
	if strings.TrimSpace(out) != "" {
		if stdout != nil {
			fmt.Fprint(stdout, out)
		} else {
			fmt.Fprint(os.Stdout, out)
		}
	}
	return err
}
