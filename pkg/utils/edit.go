package utils

import (
	"os"
	"os/exec"
)

var (
	editor     string
	editorPath string
	editorlist = []string{
		"code",
		"cursor",
		"trae",
		"vim",
		"nvim",
		"nano",
		"subl",
		"edit",
	}
)

// SetEditor sets the editor to use for editing files.
func SetEditor(e string) error {
	if e != "" {
		// Check if the provided editor exists
		if path, err := exec.LookPath(e); err == nil {
			editor = e
			editorPath = path
		} else {
			editor = e
			editorPath = e // Use as-is if not found in PATH
		}
	} else {
		// Auto-detect editor from environment or predefined list
		if envEditor := os.Getenv("EDITOR"); envEditor != "" {
			if path, err := exec.LookPath(envEditor); err == nil {
				editor = envEditor
				editorPath = path
				return nil
			}
		}
		// Search through predefined editor list
		for _, ed := range editorlist {
			if path, err := exec.LookPath(ed); err == nil {
				editor = ed
				editorPath = path
				break
			}
		}
	}
	if editor == "" {
		return os.ErrNotExist
	}
	return nil
}

// Editor returns the editor command name to use for editing files.
func Editor() (string, error) {
	err := SetEditor("")
	return editor, err
}

// HasEditor returns true if an editor is available.
func HasEditor() bool {
	editor, err := Editor()

	return editor != "" && err == nil
}
