package utils

import "testing"

func TestSetEditor(t *testing.T) {
	err := SetEditor("code")
	if err != nil {
		t.Errorf("Failed to set editor: %v", err)
	}

	err = SetEditor("")
	if err != nil {
		t.Errorf("Failed to auto-detect editor: %v", err)
	}
}

func TestEditor(t *testing.T) {
	editor, err := Editor()
	if err != nil {
		t.Errorf("Failed to get editor: %v", err)
	}

	if editor == "" {
		t.Error("Editor should not be empty")
	}
}

func TestHasEditor(t *testing.T) {
	if !HasEditor() {
		t.Error("Expected editor to be available")
	}
}
