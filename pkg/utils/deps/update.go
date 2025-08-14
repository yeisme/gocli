// Package deps provides utilities for managing Go module dependencies.
package deps

import "github.com/yeisme/gocli/pkg/tools"

// RunGoUpdate executes `go get -u` to update Go module dependencies.
func RunGoUpdate(args []string) (string, error) {
	if args == nil {
		args = []string{"./..."} // Default to updating all dependencies
	}
	output, err := tools.NewExecutor("go", append([]string{"get", "-u"}, args...)...).Output()
	if err != nil {
		return "", err
	}
	return output, nil
}
