package parser

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStderr runs fn with os.Stderr redirected and returns what it wrote.
// The parser package must write nothing there: while it printed during the
// parse, a caller could not have the answer without the warnings, and spekk
// next emitted each one twice.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stderr
	os.Stderr = w
	fn()
	os.Stderr = saved
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// brokenTree builds one specs/ tree holding each kind of skip the parser
// records, plus a spec with no assertions/ directory, which is not a skip.
func brokenTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// A malformed spec file. It carries no assertions: an assertion naming a
	// spec that failed to parse is a hard error, not a warning.
	writeFile(t, filepath.Join(dir, "bad-spec", "bad-spec.md"),
		"---\nid: bad-spec\ncreated: nonsense\npriority: 1\n---\n# Bad\n")

	// A malformed assertion file, and a duplicate id beside it.
	writeFile(t, filepath.Join(dir, "good", "good.md"),
		"---\nid: good\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Good\n")
	writeFile(t, filepath.Join(dir, "good", "assertions", "a.md"),
		"---\nid: a\nparent: good\ncreated: 2026-01-01T00:00:00Z\npriority: 9\nstatus: not_started\n---\n# A\n")
	writeFile(t, filepath.Join(dir, "good", "assertions", "b.md"),
		"---\nid: dup\nparent: good\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n---\n# B\n")
	writeFile(t, filepath.Join(dir, "good", "assertions", "c.md"),
		"---\nid: dup\nparent: good\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n---\n# C\n")

	// Assertion files with no main spec file: the whole directory is dropped.
	writeFile(t, filepath.Join(dir, "orphan", "assertions", "o.md"),
		"---\nid: o\nparent: orphan\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n---\n# O\n")

	// A drafted spec with no assertions yet. This is not a skip.
	writeFile(t, filepath.Join(dir, "lonely", "lonely.md"),
		"---\nid: lonely\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Lonely\n")

	return dir
}

// The parser records what it skipped instead of printing it. While it printed
// during the parse, spekk next emitted every warning twice, because it parses
// once for the index and once to answer.
func TestParseAllSpecsCollectsWarningsAndPrintsNothing(t *testing.T) {
	dir := brokenTree(t)

	var result *ParseResult
	var parseErr error
	printed := captureStderr(t, func() {
		result, parseErr = ParseAllSpecs(dir)
	})

	if parseErr != nil {
		t.Fatalf("unexpected error: %v", parseErr)
	}
	if printed != "" {
		t.Errorf("the parser must write nothing to stderr, got:\n%s", printed)
	}

	joined := strings.Join(result.Warnings, "\n")
	for _, want := range []string{
		"Skipping malformed spec file",
		"Skipping malformed assertion file",
		"Duplicate assertion id",
		"has no main spec file",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected a warning containing %q, got:\n%s", want, joined)
		}
	}
}

// A spec directory with no assertions/ directory is a spec nobody has broken
// into assertions yet. The spec parses and is returned; the warning that once
// called this a skip was false on both counts.
func TestSpecWithNoAssertionsDirectoryIsNotAWarning(t *testing.T) {
	result, err := ParseAllSpecs(brokenTree(t))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(result.Warnings, "\n"), "assertions/ directory") {
		t.Errorf("a spec with no assertions/ directory must not warn, got:\n%s",
			strings.Join(result.Warnings, "\n"))
	}
	var found bool
	for _, s := range result.Specs {
		if s.ID == "lonely" {
			found = true
		}
	}
	if !found {
		t.Error("a spec with no assertions/ directory must still be returned")
	}
}

// The summary replaces one line per skipped file with one line for all of them,
// and stays silent on a clean tree so a caller can print it unconditionally.
func TestWarningSummary(t *testing.T) {
	cases := []struct {
		warnings []string
		want     string
	}{
		{nil, ""},
		{[]string{"a"}, `Warning: 1 spec file skipped and missing from the queue. Run "spekk validate" for detail.`},
		{[]string{"a", "b", "c"}, `Warning: 3 spec files skipped and missing from the queue. Run "spekk validate" for detail.`},
	}
	for _, c := range cases {
		got := (&ParseResult{Warnings: c.warnings}).WarningSummary()
		if got != c.want {
			t.Errorf("for %d warnings:\n got %q\nwant %q", len(c.warnings), got, c.want)
		}
	}
}
