package tools

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yeisme/gocli/pkg/types"
	"github.com/yeisme/gocli/pkg/utils"
)

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

// InstallGitTools installs Git tools by cloning repositories and running build commands
func InstallGitTools(tools []types.GitTool) error {
	if len(tools) == 0 {
		utils.Info("No Git tools to install")
		return nil
	}

	utils.Progress("Installing %d Git tools...", len(tools))

	for i, tool := range tools {
		if !utils.IsQuiet() {
			utils.Progress("(%d/%d) Installing Git tool: %s", i+1, len(tools), tool.Name)
		}

		// Clone the repository
		targetPath, err := gitClone(tool)
		if err != nil {
			return fmt.Errorf("failed to clone %s: %w", tool.Name, err)
		}

		// Build the tool using the specified build system
		if err := buildGitTool(tool, targetPath); err != nil {
			utils.Warning("Failed to build %s: %v", tool.Name, err)
			continue
		}

		// Copy executable to ~/.gocli/bin if path is specified
		if tool.Path != "" {
			if err := copyExecutableToGoCliBin(tool, targetPath); err != nil {
				utils.Warning("Failed to copy executable for %s: %v", tool.Name, err)
				continue
			}
		}

		utils.Complete("%s installed successfully", tool.Name)
	}

	return nil
}

// buildGitTool builds a Git tool using the specified build system type
func buildGitTool(tool types.GitTool, targetPath string) error {
	switch tool.Type {
	case "make", "makefile":
		return buildWithMakefile(tool, targetPath)
	case "task", "taskfile":
		return buildWithTaskfile(tool, targetPath)
	case "just", "justfile":
		return buildWithJustfile(tool, targetPath)
	default:
		return fmt.Errorf("unsupported build system type: %s", tool.Type)
	}
}

// buildWithMakefile builds using Makefile
func buildWithMakefile(tool types.GitTool, targetPath string) error {
	// Check if make is available
	makePath, err := exec.LookPath("make")
	if err != nil {
		return fmt.Errorf("make not found in PATH")
	}

	target := ""
	if tool.Recipe != "" {
		target = tool.Recipe
	}

	cmd := exec.Command(makePath, target)
	cmd.Dir = targetPath

	if utils.IsVerbose() {
		utils.Debug("Running: make %s in %s", target, targetPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("make build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// buildWithTaskfile builds using Taskfile
func buildWithTaskfile(tool types.GitTool, targetPath string) error {
	// Check if task is available
	taskPath, err := exec.LookPath("task")
	if err != nil {
		return fmt.Errorf("task not found in PATH")
	}

	task := ""
	if tool.Recipe != "" {
		task = tool.Recipe
	}

	cmd := exec.Command(taskPath, task)
	cmd.Dir = targetPath

	if utils.IsVerbose() {
		utils.Debug("Running: task %s in %s", task, targetPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("task build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// buildWithJustfile builds using Justfile
func buildWithJustfile(tool types.GitTool, targetPath string) error {
	// Check if just is available
	justPath, err := exec.LookPath("just")
	if err != nil {
		return fmt.Errorf("just not found in PATH")
	}

	recipe := "default"
	if tool.Recipe != "" {
		recipe = tool.Recipe
	}

	cmd := exec.Command(justPath, recipe)
	cmd.Dir = targetPath

	if utils.IsVerbose() {
		utils.Debug("Running: just %s in %s", recipe, targetPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("just build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// gitClone clones a Git repository using the provided URL
func gitClone(tool types.GitTool) (string, error) {
	// Extract repository name from URL
	urlParts := strings.Split(strings.TrimSuffix(tool.URL, ".git"), "/")
	repoPath := urlParts[len(urlParts)-1]

	// Build target path: ~/.gocli/git/<name>
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	targetPath := filepath.Join(homeDir, ".gocli", "git", repoPath)

	// Check if directory already exists
	if _, err := os.Stat(targetPath); err == nil {
		if utils.IsVerbose() {
			utils.Debug("%s already exists, skipping clone", targetPath)
		}
		return targetPath, nil
	}

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create parent directory: %w", err)
	}

	cmd := exec.Command("git", "clone", "--depth", "1", tool.URL, targetPath)
	if utils.IsVerbose() {
		utils.Debug("Running: git clone %s %s", tool.URL, targetPath)
	}

	// Set output to capture errors
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to clone %s: %w\nOutput: %s", tool.Name, err, string(output))
	}

	if utils.IsVerbose() {
		utils.Complete("%s cloned successfully to %s", tool.Name, targetPath)
	}
	return targetPath, nil
}

// copyExecutableToGoCliBin copies the built executable to ~/.gocli/bin directory
func copyExecutableToGoCliBin(tool types.GitTool, targetPath string) error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create ~/.gocli/bin directory
	binDir := filepath.Join(homeDir, ".gocli", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Build source path from tool.Path
	sourcePath := filepath.Join(targetPath, tool.Path)

	// Find executable files in the source directory
	executables, err := findExecutableFiles(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to find executable files in %s: %w", sourcePath, err)
	}

	if len(executables) == 0 {
		return fmt.Errorf("no executable files found in %s", sourcePath)
	}

	// Copy each executable to ~/.gocli/bin
	for _, execFile := range executables {
		srcFile := filepath.Join(sourcePath, execFile)
		dstFile := filepath.Join(binDir, execFile)

		if utils.IsVerbose() {
			utils.Debug("Copying %s to %s", srcFile, dstFile)
		}

		if err := copyFile(srcFile, dstFile); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", srcFile, dstFile, err)
		}

		// Make executable on Unix-like systems
		if runtime.GOOS != "windows" {
			if err := os.Chmod(dstFile, 0755); err != nil {
				return fmt.Errorf("failed to set executable permissions on %s: %w", dstFile, err)
			}
		}

		utils.Complete("Copied %s to ~/.gocli/bin/", execFile)
	}

	return nil
}

// findExecutableFiles finds executable files in the given directory
func findExecutableFiles(dir string) ([]string, error) {
	var executables []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Check if it's an executable based on file extension and permissions
		if isExecutableFile(filepath.Join(dir, filename)) {
			executables = append(executables, filename)
		}
	}

	return executables, nil
}

// isExecutableFile checks if a file is executable
func isExecutableFile(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	// On Windows, check for .exe extension
	if runtime.GOOS == "windows" {
		return strings.HasSuffix(strings.ToLower(filePath), ".exe")
	}

	// On Unix-like systems, check execute permissions
	mode := info.Mode()
	return mode&0111 != 0 && !info.IsDir()
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Sync to ensure data is written to disk
	return destFile.Sync()
}
