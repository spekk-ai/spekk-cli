package parser

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStderr runs fn with os.Stderr redirected and returns what it wrote.
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

// The accepted list covers two conventions, because a project selects one.
// Before this, the list accepted gitflow only, so each assertion in a
// conventional-commits project got a warning. This repository gave 73.
func TestStandardBranchesWarnForNeitherConvention(t *testing.T) {
	accepted := []string{
		// standalone
		"main", "master", "develop",
		// gitflow
		"feature/list-filter", "bugfix/parser-crash", "hotfix/urgent", "release/1.22.0",
		// conventional commits
		"feat/list-filter", "fix/parser-crash", "chore/cleanup", "docs/ci-page",
		"refactor/index", "test/coverage", "perf/query", "build/release",
		"ci/workflows", "style/format",
	}
	for _, branch := range accepted {
		out := captureStderr(t, func() {
			if err := validateBranch(branch, "specs/demo/assertions/a.md"); err != nil {
				t.Errorf("validateBranch(%q) returned an error: %v", branch, err)
			}
		})
		if out != "" {
			t.Errorf("%q must not warn, got:\n%s", branch, out)
		}
	}
}

// The warning keeps its function. A value in neither convention is usually a
// bare name or a spelling error, and such a field usually names no branch.
// The first two examples below come from this repository.
func TestBareBranchNamesStillWarn(t *testing.T) {
	for _, branch := range []string{"temporary-target", "observer-reimpl", "feture/typo"} {
		out := captureStderr(t, func() {
			if err := validateBranch(branch, "specs/demo/assertions/a.md"); err != nil {
				t.Errorf("validateBranch(%q) returned an error: %v", branch, err)
			}
		})
		if !strings.Contains(out, "non-standard pattern") {
			t.Errorf("%q must warn, got: %q", branch, out)
		}
	}
}

// The old message listed four values, and the pattern accepted seven, so
// develop, master, and release/ were accepted and not documented. Both now
// come from the same two lists. This test fails if they disagree again.
func TestWarningNamesEveryAcceptedValue(t *testing.T) {
	out := captureStderr(t, func() {
		_ = validateBranch("temporary-target", "specs/demo/assertions/a.md")
	})
	for _, name := range standardBranchNames {
		if !strings.Contains(out, name) {
			t.Errorf("warning must name the accepted branch %q, got:\n%s", name, out)
		}
	}
	for _, typ := range standardBranchTypes {
		if !strings.Contains(out, typ) {
			t.Errorf("warning must name the accepted type %q, got:\n%s", typ, out)
		}
	}
}

// Git permits a dot in a branch name, and a release branch usually carries a
// version. Git's rules for dots must continue to apply.
func TestBranchDotRules(t *testing.T) {
	for _, branch := range []string{"release/1.22.0", "feat/v2.0-spike"} {
		if err := validateBranch(branch, "specs/demo/assertions/a.md"); err != nil {
			t.Errorf("validateBranch(%q) must be accepted, got: %v", branch, err)
		}
	}
	for _, branch := range []string{"release/1..22", "feat/thing.", "feat/thing.lock"} {
		if err := validateBranch(branch, "specs/demo/assertions/a.md"); err == nil {
			t.Errorf("validateBranch(%q) must be refused: git refuses it too", branch)
		}
	}
}

// A short type must not match inside a longer name. A match on a substring
// instead of a prefix would accept an invalid value such as "myfeat/x".
func TestBranchTypesMatchAsAPrefixOnly(t *testing.T) {
	for _, branch := range []string{"myfeat/thing", "notfix/thing", "xmain"} {
		out := captureStderr(t, func() {
			_ = validateBranch(branch, "specs/demo/assertions/a.md")
		})
		if !strings.Contains(out, "non-standard pattern") {
			t.Errorf("%q must warn: the type is a prefix, not a substring; got: %q", branch, out)
		}
	}
}
