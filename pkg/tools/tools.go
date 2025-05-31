package tools

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

func ToolsCheck(dev, goTools, gitTools, customTools bool, cfg types.Tools) error {
	hasErrors := false

	if dev && len(cfg.Dev) > 0 {
		utils.SubHeader("Development Tools")
		if err := checkDevTools(cfg.Dev); err != nil {
			utils.Error("Development tools check failed: %v", err)
			hasErrors = true
		}
		fmt.Println()
	}

	if goTools && len(cfg.Go) > 0 {
		utils.SubHeader("Go Tools")
		if err := checkGoTools(cfg.Go); err != nil {
			utils.Warning("Some Go tools are missing: %v", err)
		}
		fmt.Println()
	}

	if gitTools && len(cfg.Git) > 0 {
		utils.SubHeader("Git Tools")
		if err := checkGitTools(cfg.Git); err != nil {
			utils.Warning("Git tools need manual installation: %v", err)
		}
		fmt.Println()
	}

	if customTools && len(cfg.Custom) > 0 {
		utils.SubHeader("Custom Tools")
		if err := checkCustomTools(cfg.Custom); err != nil {
			utils.Warning("Custom tools need manual setup: %v", err)
		}
	}

	if hasErrors {
		return fmt.Errorf("some required tools are missing")
	}

	return nil
}

func checkDevTools(tools []types.DevTool) error {
	missingTools := []string{}

	for _, tool := range tools {
		if err := checkToolInPath(tool.Name, tool.Version); err != nil {
			utils.ErrorToStderr("%s: %v", tool.Name, err)
			missingTools = append(missingTools, tool.Name)
		} else {
			utils.Complete("%s is available", tool.Name)
		}
	}

	if len(missingTools) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missingTools, ", "))
	}

	return nil
}

func checkGoTools(tools []types.GoTool) error {
	missingTools := []string{}

	for _, tool := range tools {
		if err := checkToolInPath(tool.Bin, ""); err != nil {
			utils.Warning("%s not found (install: go install %s)", tool.Name, tool.URL)
			missingTools = append(missingTools, tool.Name)
		} else {
			utils.Complete("%s is available", tool.Name)
		}
	}

	if len(missingTools) > 0 {
		return fmt.Errorf("missing Go tools: %s", strings.Join(missingTools, ", "))
	}

	return nil
}

func checkGitTools(tools []types.GitTool) error {
	for _, tool := range tools {
		utils.Info("📋 %s: %s", tool.Name, tool.Description)
		utils.ListItem("Repository: %s", tool.URL)
		utils.ListItem("Build type: %s", tool.Type)
		if tool.Recipe != "" {
			utils.ListItem("Build recipe: %s", tool.Recipe)
		}
		fmt.Println()
	}
	return nil
}

func checkCustomTools(tools []types.CustomTool) error {
	for _, tool := range tools {
		utils.Info("🔧 %s", tool.Name)
		utils.ListItem("Command: %s", tool.Cmd)
		if len(tool.Needs) > 0 {
			utils.ListItem("Dependencies: %s", strings.Join(tool.Needs, ", "))
		}
		fmt.Println()
	}
	return nil
}

func checkToolInPath(toolName, version string) error {
	_, err := exec.LookPath(toolName)
	if err != nil {
		return fmt.Errorf("not found in PATH")
	}

	// If version is specified, check it
	if version != "" && !strings.HasPrefix(version, ">=") {
		// For now, just check if tool exists
		// Version checking can be implemented later based on specific tool output
		utils.Debug("Version checking for %s: %s (not implemented)", toolName, version)
	}

	return nil
}

func InstallGoTools(tools []types.GoTool) error {
	if len(tools) == 0 {
		utils.Info("No Go tools to install")
		return nil
	}

	utils.Progress("Installing %d Go tools...", len(tools))

	for i, tool := range tools {
		utils.Progress("(%d/%d) Installing %s...", i+1, len(tools), tool.Name)

		args := []string{"install"}
		args = append(args, tool.Flags...)
		args = append(args, tool.URL)

		cmd := exec.Command("go", args...)
		if utils.IsVerbose() {
			utils.Debug("Running: go %s", strings.Join(args, " "))
		}

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w", tool.Name, err)
		}

		utils.Complete("%s installed successfully", tool.Name)
	}

	return nil
}

// InstallAllGoTools installs all Go tools from configuration
func InstallAllGoTools(cfg types.Tools) error {
	return InstallGoTools(cfg.Go)
}

// CheckAllTools checks all tool categories
func CheckAllTools(cfg types.Tools) error {
	return ToolsCheck(true, true, true, true, cfg)
}

// CheckToolAvailability checks if a specific tool is available
func CheckToolAvailability(toolName string) bool {
	_, err := exec.LookPath(toolName)
	return err == nil
}
