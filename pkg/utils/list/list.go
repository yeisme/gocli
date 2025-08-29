// Package list like `go list` lists Go packages in the current module.
package list

import (
	"context"

	"github.com/yeisme/gocli/pkg/utils/executor"
)

// RunGoList executes the `go list` command with the provided arguments and options.
func RunGoList(ctx context.Context, opts struct {
	JSON, Test bool
}, args []string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	listArgs := []string{}
	if opts.JSON {
		listArgs = append(listArgs, "-json")
	}
	if opts.Test {
		listArgs = append(listArgs, "-test")
	}

	listArgs = append(listArgs, args...)

	args = append([]string{"list"}, listArgs...)

	goList := executor.NewExecutor("go", args...)
	output, err := goList.Output()
	if err != nil {
		return "", err
	}
	return output, nil
}
