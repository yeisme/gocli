package fsop

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListAllSubdirectories(t *testing.T) {
	root := t.TempDir()
	subdir1 := filepath.Join(root, "subdir1")
	subdir2 := filepath.Join(root, "subdir2")
	subsubdir := filepath.Join(subdir1, "subsubdir")

	// Create subdirectories
	if err := os.MkdirAll(subsubdir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	if err := os.Mkdir(subdir2, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Call the function to test
	subdirs, err := ListAllSubdirectories(root)
	if err != nil {
		t.Fatalf("ListAllSubdirectories returned an error: %v", err)
	}

	// Check if the expected subdirectories are returned
	expected := []string{subdir1, subsubdir, subdir2}
	if !reflect.DeepEqual(subdirs, expected) {
		t.Errorf("Expected %v, got %v", expected, subdirs)
	}
}
