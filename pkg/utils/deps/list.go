package deps

import "github.com/yeisme/gocli/pkg/tools"

// RunGoModList executes `go list -u -m all` to list all module dependencies with updates.
func RunGoModList(args []string, option struct {
	JSON   bool
	Update bool
}) (string, error) {
	// Always start with base command
	base := []string{"list", "-m"}
	if option.JSON {
		base = append(base, "-json")
	}
	if option.Update {
		base = append(base, "-u")
	}
	// Treat incoming args as targets/patterns
	if len(args) == 0 {
		base = append(base, "all")
	} else {
		base = append(base, args...)
	}

	output, err := tools.NewExecutor("go", base...).Output()
	if err != nil {
		return "", err
	}
	return output, nil

}
