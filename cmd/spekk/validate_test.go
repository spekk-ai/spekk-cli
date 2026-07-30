package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeCleanValidateFixture creates a temp specs/ tree that is valid under the
// new `validate` invariants specifically: unlike the shared makeTmpSpecs
// fixture (used by list_test.go), the parent spec here has no `status` field
// at all, since an explicit `status: not_started` on a parent is exactly the
// footgun validate's parent-status check exists to catch.
func makeCleanValidateFixture(t *testing.T) string {
	t.Helper()
	specsDir := t.TempDir()
	specDir := filepath.Join(specsDir, "my-spec")
	assertionsDir := filepath.Join(specDir, "assertions")
	if err := os.MkdirAll(assertionsDir, 0o755); err != nil {
		t.Fatalf("makeCleanValidateFixture: create assertions dir: %v", err)
	}
	specContent := `---
id: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
---
# My Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "my-spec.md"), []byte(specContent), 0o644); err != nil {
		t.Fatalf("makeCleanValidateFixture: write spec: %v", err)
	}
	assertionContent := `---
id: my-assertion
parent: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: not_started
---
# My Assertion
`
	if err := os.WriteFile(filepath.Join(assertionsDir, "my-assertion.md"), []byte(assertionContent), 0o644); err != nil {
		t.Fatalf("makeCleanValidateFixture: write assertion: %v", err)
	}
	return specsDir
}

// TestExecValidate_CleanTree_ExitZero exercises the CLI wiring (flag parsing,
// specsDir resolution, stdout report) over a minimal well-formed fixture.
func TestExecValidate_CleanTree_ExitZero(t *testing.T) {
	specsDir := makeCleanValidateFixture(t)
	var stdout bytes.Buffer

	code := execValidate(nil, &stdout, specsDir)

	if code != 0 {
		t.Errorf("expected exit code 0, got %d; stdout: %q", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "OK") {
		t.Errorf("expected a success summary on stdout, got: %q", stdout.String())
	}
}

// TestExecValidate_BrokenTree_ExitNonZero drives the same CLI path over a
// fixture with an in_progress assertion missing locked-by, confirming the
// failure surfaces on stdout with a non-zero exit code.
func TestExecValidate_BrokenTree_ExitNonZero(t *testing.T) {
	specsDir := makeCleanValidateFixture(t)
	assertionPath := filepath.Join(specsDir, "my-spec", "assertions", "my-assertion.md")
	broken := `---
id: my-assertion
parent: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: in_progress
---
# My Assertion
`
	if err := os.WriteFile(assertionPath, []byte(broken), 0o644); err != nil {
		t.Fatalf("writing broken fixture: %v", err)
	}

	var stdout bytes.Buffer
	code := execValidate(nil, &stdout, specsDir)

	if code == 0 {
		t.Fatal("expected non-zero exit code for a broken tree")
	}
	if !strings.Contains(stdout.String(), "locked-by is missing") {
		t.Errorf("expected failure line about locked-by, got: %q", stdout.String())
	}
}

// TestExecValidate_Help_ExitZero confirms --help short-circuits before any
// specsDir resolution.
func TestExecValidate_Help_ExitZero(t *testing.T) {
	var stdout bytes.Buffer
	code := execValidate([]string{"--help"}, &stdout, "")
	if code != 0 {
		t.Errorf("expected exit code 0 for --help, got %d", code)
	}
	if !strings.Contains(stdout.String(), "spekk validate") {
		t.Errorf("expected usage text, got: %q", stdout.String())
	}
}
