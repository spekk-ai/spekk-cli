package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every skip the parser records must be reachable through validate, because
// the one-line summary that replaced the per-file warnings tells the reader to
// run it for the detail.

func writeAt(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func failureMessages(t *testing.T, specsDir string) string {
	t.Helper()
	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var b strings.Builder
	for _, f := range result.Failures {
		b.WriteString(f.String())
		b.WriteString("\n")
	}
	return b.String()
}

// The parser drops the whole directory, so every assertion in it is lost.
// Before this, validate reported only the downstream "parent not found", and
// only when an assertion inside happened to name that parent.
func TestAssertionsWithNoMainSpecFileFail(t *testing.T) {
	dir := t.TempDir()
	writeAt(t, filepath.Join(dir, "orphan", "assertions", "o.md"),
		"---\nid: o\nparent: orphan\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n---\n# O\n")

	got := failureMessages(t, dir)
	if !strings.Contains(got, "spec directory has assertion files but no main spec file") {
		t.Errorf("expected the missing-spec-file failure, got:\n%s", got)
	}
}

// A path named assertions that is a regular file makes the parser drop the
// spec's assertions in silence.
func TestAssertionsPathThatIsNotADirectoryFails(t *testing.T) {
	dir := t.TempDir()
	writeAt(t, filepath.Join(dir, "demo", "demo.md"),
		"---\nid: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Demo\n")
	writeAt(t, filepath.Join(dir, "demo", "assertions"), "not a directory\n")

	got := failureMessages(t, dir)
	if !strings.Contains(got, "specs/demo/assertions: expected a directory") {
		t.Errorf("expected the not-a-directory failure, got:\n%s", got)
	}
}

// A spec with no assertions/ directory parses correctly and is a normal spec
// nobody has broken into assertions yet. The parser's warning that called this
// a skip was false, and validate must not reintroduce it as a failure.
func TestSpecWithNoAssertionsDirectoryPasses(t *testing.T) {
	dir := t.TempDir()
	writeAt(t, filepath.Join(dir, "lonely", "lonely.md"),
		"---\nid: lonely\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Lonely\n")

	result, err := Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed() {
		t.Errorf("a spec with no assertions must pass, got: %v", result.Failures)
	}
	if result.SpecCount != 1 {
		t.Errorf("the spec must still be counted, got %d", result.SpecCount)
	}
}

// A directory holding no file with frontmatter is not a spec directory, so a
// missing main spec file there is not a problem.
func TestDirectoryWithNoFrontmatterIsIgnored(t *testing.T) {
	dir := t.TempDir()
	writeAt(t, filepath.Join(dir, "notes", "assertions", "readme.md"), "# Just notes\n")

	result, err := Run(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed() {
		t.Errorf("a directory with no frontmatter must be ignored, got: %v", result.Failures)
	}
}
