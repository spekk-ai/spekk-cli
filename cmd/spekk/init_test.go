package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunInit_FreshWritesManagedReadme confirms that running spekk init in
// an empty directory creates specs/README.md containing the managed region
// (with both fence markers) and ending in exactly one trailing newline.
func TestRunInit_FreshWritesManagedReadme(t *testing.T) {
	dir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })

	runInit(nil)

	readmePath := filepath.Join(dir, "specs", "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", readmePath, err)
	}
	got := string(content)

	if !strings.Contains(got, readmeManagedBeginMarker) || !strings.Contains(got, readmeManagedEndMarker) {
		t.Errorf("expected specs/README.md to contain both fence markers, got:\n%s", got)
	}
	if strings.HasSuffix(got, "\n\n") || !strings.HasSuffix(got, "\n") {
		t.Errorf("expected specs/README.md to end with exactly one trailing newline, got tail %q", got[len(got)-10:])
	}
}
