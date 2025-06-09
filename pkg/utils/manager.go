package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetManagerFile(manager string) string {
	switch manager {
	case "task":
		return "Taskfile"
	case "just":
		return "Justfile"
	case "make":
		return "Makefile"
	default:
		return "Taskfile"
	}
}

func GetManager(manager string) string {
	switch manager {
	case "task":
		return "task"
	case "just":
		return "just"
	case "make":
		return "make"
	default:
		return "task"
	}
}

func GetProjectName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return filepath.Base(wd)
}

func GetAuthor() string {
	author, err := exec.Command("git", "config", "--get", "user.name").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(author))
}

func GetPluginsHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gocli", "plugins")
}
