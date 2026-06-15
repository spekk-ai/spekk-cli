package parser

import "testing"

// The exported *Content wrappers must reuse the existing per-file parse logic so
// callers (e.g. the cross-branch explorer) can parse file bytes read out of a
// git ref without the file existing on disk.

func TestParseSpecContent_ReusesParser(t *testing.T) {
	content := "---\nid: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 2\nstatus: done\nbranch: feature/x\n---\n# My Spec Title\n\nBody.\n"

	spec, err := ParseSpecContent("specs/my-spec/my-spec.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.ID != "my-spec" {
		t.Errorf("ID = %q, want %q", spec.ID, "my-spec")
	}
	if spec.Status != "done" {
		t.Errorf("Status = %q, want %q", spec.Status, "done")
	}
	if spec.Branch != "feature/x" {
		t.Errorf("Branch = %q, want %q", spec.Branch, "feature/x")
	}
	if spec.Title != "My Spec Title" {
		t.Errorf("Title = %q, want %q", spec.Title, "My Spec Title")
	}
	if spec.File != "specs/my-spec/my-spec.md" {
		t.Errorf("File = %q, want relFilePath passthrough", spec.File)
	}
}

func TestParseAssertionContent_ReusesParser(t *testing.T) {
	content := "---\nid: my-assert\nparent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: in_progress\n---\n# Assertion Title\n"

	a, err := ParseAssertionContent("specs/my-spec/assertions/my-assert.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID != "my-assert" {
		t.Errorf("ID = %q, want %q", a.ID, "my-assert")
	}
	if a.Parent != "my-spec" {
		t.Errorf("Parent = %q, want %q", a.Parent, "my-spec")
	}
	if a.Status != "in_progress" {
		t.Errorf("Status = %q, want %q", a.Status, "in_progress")
	}
	if a.Title != "Assertion Title" {
		t.Errorf("Title = %q, want %q", a.Title, "Assertion Title")
	}
}

func TestParseSpecContent_PropagatesParseError(t *testing.T) {
	// Missing required 'id' must surface as an error from the wrapper, exactly
	// as the underlying parser reports it — errors are not swallowed.
	_, err := ParseSpecContent("specs/bad/bad.md", "---\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# X\n")
	if err == nil {
		t.Fatal("expected error for missing id, got nil")
	}
}
