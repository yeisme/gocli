package utils

import (
	"os/exec"
)

var (
	editor     string
	editorlist = []string{
		"code",
		"vim",
		"nano",
		"nvim",
		"edit",
	}
)

// SetEditor sets the editor to use for editing files.
func SetEditor(e string) {
	if e != "" {
		editor = e
	} else {
		for _, ed := range editorlist {
			if _, err := exec.LookPath(ed); err == nil {
				editor = ed
				break
			}
		}
	}
}

// Editor returns the editor to use for editing files.
func Editor() string {
	if editor == "" {
		SetEditor("")
	}
	return editor
}
