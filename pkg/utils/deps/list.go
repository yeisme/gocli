package deps

import "github.com/yeisme/gocli/pkg/tools"

// RunGoModList executes `go list -u -m all` to list all module dependencies with updates.
func RunGoModList(args []string, option struct {
	JSON   bool
	Update bool
}) (string, error) {
	if args == nil {
		args = []string{"list", "-m"}
	}
	if option.JSON {
		args = append(args, "-json")
	}
	if option.Update {
		args = append(args, "-u")
	}
	args = append(args, "all")

	output, err := tools.NewExecutor("go", args...).Output()
	if err != nil {
		return "", err
	}
	return output, nil

}
