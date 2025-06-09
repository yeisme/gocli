package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func GoExec(cmdStr string) error {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmdName := parts[0]
	args := parts[1:]

	path, err := exec.LookPath(cmdName)
	if err != nil {
		return fmt.Errorf("command not found: %s", cmdName)
	}

	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func GetCurrentUser() string {
	out, err := exec.Command("git", "config", "--global", "user.name").Output()
	if err != nil {
		return "unknown"
	}
	return string(out)
}
