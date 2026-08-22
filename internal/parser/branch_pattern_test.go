package parser

import (
	"io"
	"os"
	"testing"
)

// validateBranch judges no naming convention, so any name git accepts passes.
// A list of accepted <type>/ prefixes warned on every other shape, which
// passed the typo it was meant to catch and warned on the team names it was
// not. internal/validate reads the refs instead.
func TestBranchAcceptsAnyNameGitAccepts(t *testing.T) {
	accepted := []string{
		"main", "master", "develop",
		"feature/list-filter", "feat/list-filter", "release/1.22.0",
		"dana/apx-12-retry-billing-webhook", "sam/PROJ-441",
		"temporary-target", "myfeat/thing", "feat/v2.0-spike",
	}
	for _, branch := range accepted {
		if err := validateBranch(branch, "specs/demo/assertions/a.md"); err != nil {
			t.Errorf("validateBranch(%q) must be accepted, got: %v", branch, err)
		}
	}
}

// The hard errors stay: each value below is one git itself refuses.
func TestBranchRefusesWhatGitRefuses(t *testing.T) {
	for _, branch := range []string{
		"/leading", "trailing/", "feat/thing space",
		"release/1..22", "feat/thing.", "feat/thing.lock",
	} {
		if err := validateBranch(branch, "specs/demo/assertions/a.md"); err == nil {
			t.Errorf("validateBranch(%q) must be refused: git refuses it too", branch)
		}
	}
}

// The parser owns no output decision for this field. Anything it printed here
// reached stderr on every command that reads the spec tree, including the
// spekk next of each builder iteration.
func TestBranchValidationIsSilent(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stderr
	os.Stderr = w
	for _, branch := range []string{"main", "dana/apx-12-thing", "temporary-target", "/leading"} {
		_ = validateBranch(branch, "specs/demo/assertions/a.md")
	}
	os.Stderr = saved
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("validateBranch must write nothing to stderr, got:\n%s", out)
	}
}
