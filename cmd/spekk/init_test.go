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

// TestRunInit_RerunIsByteIdentical confirms the headline idempotency
// guarantee at the CLI level: running spekk init again on an already
// initialized project regenerates the managed block in place, and a second
// re-run (with no schema change) leaves specs/README.md byte-identical.
func TestRunInit_RerunIsByteIdentical(t *testing.T) {
	dir := chdirTemp(t)
	readmePath := filepath.Join(dir, "specs", "README.md")

	runInit(nil)
	first, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("reading %s after first init: %v", readmePath, err)
	}

	runInit(nil)
	second, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("reading %s after second init: %v", readmePath, err)
	}

	if string(first) != string(second) {
		t.Fatalf("expected a re-run of spekk init to be byte-identical\nfirst:\n%q\nsecond:\n%q", first, second)
	}
}

// TestRunInit_RerunPreservesHumanProseAfterFence confirms that human prose
// appended after the managed block survives a re-run of spekk init, and
// that the managed fence is still intact afterward.
func TestRunInit_RerunPreservesHumanProseAfterFence(t *testing.T) {
	dir := chdirTemp(t)
	readmePath := filepath.Join(dir, "specs", "README.md")

	runInit(nil)

	f, err := os.OpenFile(readmePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("opening %s to append prose: %v", readmePath, err)
	}
	if _, err := f.WriteString("\n## My Notes\n\nDon't touch this.\n"); err != nil {
		f.Close()
		t.Fatalf("appending human prose: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing %s: %v", readmePath, err)
	}

	runInit(nil)

	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("reading %s after re-run: %v", readmePath, err)
	}
	got := string(content)

	if !strings.Contains(got, "Don't touch this.") {
		t.Errorf("expected human prose appended after the fence to survive a re-run, got:\n%s", got)
	}
	if !strings.Contains(got, readmeManagedBeginMarker) || !strings.Contains(got, readmeManagedEndMarker) {
		t.Errorf("expected the managed fence to still be present after a re-run, got:\n%s", got)
	}
}
