package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdirTemp creates a temp dir, chdirs into it, and restores the original
// working directory on test cleanup.
func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })
	return dir
}

// TestRunInit_FreshWritesManagedReadme confirms that running spekk init in
// an empty directory creates specs/README.md containing the managed region
// (with both fence markers) and ending in exactly one trailing newline.
func TestRunInit_FreshWritesManagedReadme(t *testing.T) {
	dir := chdirTemp(t)

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

// TestRunInit_UpgradesLegacyReadme confirms that running spekk init against
// a pre-existing specs/README.md with no managed fence (the historical
// static README, or any hand-written one) preserves that file's content and
// appends a fresh managed region, and that a second run is byte-identical.
func TestRunInit_UpgradesLegacyReadme(t *testing.T) {
	dir := chdirTemp(t)
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	readmePath := filepath.Join(specsDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(legacyStaticReadme), 0o644); err != nil {
		t.Fatalf("writing legacy README: %v", err)
	}

	runInit(nil)

	first, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("reading %s after upgrade: %v", readmePath, err)
	}
	got := string(first)

	if !strings.Contains(got, "Each spec is a folder containing a markdown file") {
		t.Errorf("expected legacy prose to survive the upgrade, got:\n%s", got)
	}
	if !strings.Contains(got, readmeManagedBeginMarker) || !strings.Contains(got, readmeManagedEndMarker) {
		t.Errorf("expected an appended managed fence, got:\n%s", got)
	}

	runInit(nil)
	second, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("reading %s after second init: %v", readmePath, err)
	}
	if string(first) != string(second) {
		t.Fatalf("expected the upgrade to converge: a second init must be byte-identical\nfirst:\n%q\nsecond:\n%q", first, second)
	}
}
