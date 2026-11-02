package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateUnknownKindWarning(t *testing.T) {
	specsDir := filepath.Join(t.TempDir(), "specs")
	assertionsDir := filepath.Join(specsDir, "sample", "assertions")
	if err := os.MkdirAll(assertionsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := "---\nid: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Sample\n"
	if err := os.WriteFile(filepath.Join(specsDir, "sample", "sample.md"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	assertion := "---\nid: property\nparent: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\nkind: invarient\n---\n# Property\n"
	if err := os.WriteFile(filepath.Join(assertionsDir, "property.md"), []byte(assertion), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := execValidate(nil, &stdout, &stderr, specsDir)
	if code != 0 || stdout.String() != "validate: 1 specs, 1 assertions OK\n" || !strings.Contains(stderr.String(), "property.md: unknown kind \"invarient\"") {
		t.Fatalf("code %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
	assertion = strings.Replace(assertion, "kind: invarient", "kind: [invariant]", 1)
	if err := os.WriteFile(filepath.Join(assertionsDir, "property.md"), []byte(assertion), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = execValidate(nil, &stdout, &stderr, specsDir)
	if code != 0 || !strings.Contains(stderr.String(), "property.md: unknown kind \"[invariant]\"") {
		t.Fatalf("list kind: code %d, stderr %q", code, stderr.String())
	}
}
