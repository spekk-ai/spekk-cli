package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// treeWithSkippedFiles writes a specs/ tree whose spec parses and whose three
// assertions are all malformed, so the parser skips each one.
func treeWithSkippedFiles(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("demo/demo.md", "---\nid: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Demo\n")
	for _, name := range []string{"a", "b", "c"} {
		write("demo/assertions/"+name+".md",
			"---\nid: "+name+"\nparent: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 9\nstatus: not_started\n---\n# "+name+"\n")
	}
	return dir
}

// Three skipped files produce one line, not three. Before this the parser
// printed one per file, and spekk next printed each twice.
func TestListPrintsOneWarningLineForManySkippedFiles(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := execList([]string{"--json"}, &stdout, &stderr, treeWithSkippedFiles(t)); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly one stderr line, got %d:\n%s", len(lines), stderr.String())
	}
	if !strings.Contains(lines[0], "3 spec files skipped") ||
		!strings.Contains(lines[0], `Run "spekk validate" for detail.`) {
		t.Errorf("unexpected summary: %q", lines[0])
	}

	// The warning must not reach stdout, or --json stops being machine-readable.
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Errorf("stdout must stay valid JSON, got error %v for:\n%s", err, stdout.String())
	}
}

// A tree with nothing to report says nothing, so a caller prints the summary
// unconditionally without adding noise to a healthy project.
func TestCleanTreeWritesNothingToStderr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo", "demo.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\nid: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	execList([]string{"--json"}, &stdout, &stderr, dir)
	if stderr.Len() != 0 {
		t.Errorf("a clean tree must write nothing to stderr, got:\n%s", stderr.String())
	}
}
