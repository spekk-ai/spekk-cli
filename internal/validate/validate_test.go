package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSpec writes a minimal well-formed spec file at specs/<id>/<id>.md,
// with extra frontmatter lines (e.g. "status: draft") inserted before the
// closing "---".
func writeSpec(t *testing.T, specsDir, id string, extraFrontmatter string) {
	t.Helper()
	dir := filepath.Join(specsDir, id)
	if err := os.MkdirAll(filepath.Join(dir, "assertions"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "---\n" +
		"id: " + id + "\n" +
		"created: 2026-01-01T00:00:00Z\n" +
		"priority: 1\n" +
		extraFrontmatter +
		"---\n# " + id + "\n"
	if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
}

// writeAssertion writes an assertion file at specs/<spec>/assertions/<id>.md
// with the given frontmatter body (id/parent/created/priority plus whatever
// extra fields the test needs).
func writeAssertion(t *testing.T, specsDir, spec, filename, frontmatter string) {
	t.Helper()
	dir := filepath.Join(specsDir, spec, "assertions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "---\n" + frontmatter + "---\n# " + filename + "\n"
	if err := os.WriteFile(filepath.Join(dir, filename+".md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write assertion: %v", err)
	}
}

func hasFailureContaining(t *testing.T, r *Result, substr string) {
	t.Helper()
	for _, f := range r.Failures {
		if strings.Contains(f.String(), substr) {
			return
		}
	}
	t.Errorf("expected a failure containing %q, got: %v", substr, r.Failures)
}

func TestRun_AllValidTree_PassesWithZeroExit(t *testing.T) {
	specsDir := t.TempDir()
	writeSpec(t, specsDir, "my-spec", "")
	writeAssertion(t, specsDir, "my-spec", "my-assertion",
		"id: my-assertion\nparent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n")

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed() {
		t.Fatalf("expected a clean tree to pass, got failures: %v", result.Failures)
	}
	if result.SpecCount != 1 || result.AssertionCount != 1 {
		t.Errorf("expected 1 spec / 1 assertion, got %d/%d", result.SpecCount, result.AssertionCount)
	}
}

// A lock says a builder holds the assertion now. An assertion somebody started
// and nobody holds is a real state, and it is the state a crashed builder
// leaves behind. Demanding a lock here forced a coach to invent one, because
// no CLI command mints a lock.
func TestRun_InProgressWithoutLockedBy_Passes(t *testing.T) {
	specsDir := t.TempDir()
	writeSpec(t, specsDir, "my-spec", "")
	writeAssertion(t, specsDir, "my-spec", "my-assertion",
		"id: my-assertion\nparent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: in_progress\n")

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed() {
		t.Fatalf("an unlocked in_progress assertion must pass, got: %v", result.Failures)
	}
}

func TestRun_DoneWithLockedBy_Fails(t *testing.T) {
	specsDir := t.TempDir()
	writeSpec(t, specsDir, "my-spec", "")
	writeAssertion(t, specsDir, "my-spec", "my-assertion",
		"id: my-assertion\nparent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: done\nlocked-by: builder-host-1-1700000000\n")

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed() {
		t.Fatal("expected failure for done assertion with locked-by set")
	}
	hasFailureContaining(t, result, "locked-by is set")
}

func TestRun_MalformedAssertion_FailsInsteadOfSkipping(t *testing.T) {
	specsDir := t.TempDir()
	writeSpec(t, specsDir, "my-spec", "")
	// priority 4 is out of range (1-3) and status is invalid; the lenient
	// parser would skip this with a warning and keep going.
	writeAssertion(t, specsDir, "my-spec", "bad-assertion",
		"id: bad-assertion\nparent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 4\nstatus: bogus\n")

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed() {
		t.Fatal("expected malformed assertion to be a hard failure, not silently skipped")
	}
	hasFailureContaining(t, result, "bad-assertion.md")
}

func TestRun_DuplicateAssertionID_Fails(t *testing.T) {
	specsDir := t.TempDir()
	writeSpec(t, specsDir, "my-spec", "")
	body := "id: dup-assertion\nparent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n"
	writeAssertion(t, specsDir, "my-spec", "dup-assertion-a", body)
	// Same id declared in a second file — the lenient parser warns-and-skips
	// the second; validate must fail.
	writeAssertion(t, specsDir, "my-spec", "dup-assertion-b",
		strings.Replace(body, "created: 2026-01-01T00:00:00Z", "created: 2026-01-02T00:00:00Z", 1))

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed() {
		t.Fatal("expected duplicate assertion id to fail")
	}
	hasFailureContaining(t, result, "duplicate assertion id")
}

func TestRun_ParentStatusDone_Fails(t *testing.T) {
	specsDir := t.TempDir()
	writeSpec(t, specsDir, "my-spec", "status: done\n")
	writeAssertion(t, specsDir, "my-spec", "my-assertion",
		"id: my-assertion\nparent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n")

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed() {
		t.Fatal("expected parent spec with status: done to fail")
	}
	hasFailureContaining(t, result, "disallowed status field")
}

func TestRun_ParentStatusDraft_Passes(t *testing.T) {
	specsDir := t.TempDir()
	writeSpec(t, specsDir, "my-spec", "status: draft\n")
	writeAssertion(t, specsDir, "my-spec", "my-assertion",
		"id: my-assertion\nparent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: draft\n")

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed() {
		t.Fatalf("expected parent spec with status: draft to pass, got: %v", result.Failures)
	}
}

// TestRun_ParentStatusAbsentVsExplicitNotStarted is the load-bearing
// regression test for the latent-bug fix: ParseSpecContent defaults an
// *absent* status to "not_started", so the check must distinguish the two
// using the raw frontmatter text, not the parsed Spec.Status.
func TestRun_ParentStatusAbsentVsExplicitNotStarted(t *testing.T) {
	specsDir := t.TempDir()
	writeSpec(t, specsDir, "absent-status", "") // no status field at all
	writeSpec(t, specsDir, "explicit-not-started", "status: not_started\n")

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, f := range result.Failures {
		if strings.HasPrefix(f.File, "specs/absent-status/") {
			t.Errorf("absent status field must pass, got failure: %s", f)
		}
	}
	hasFailureContaining(t, result, "specs/explicit-not-started/explicit-not-started.md")
}

func TestRun_DanglingDependsOn_Fails(t *testing.T) {
	specsDir := t.TempDir()
	writeSpec(t, specsDir, "my-spec", "")
	writeAssertion(t, specsDir, "my-spec", "my-assertion",
		"id: my-assertion\nparent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\ndepends-on: nonexistent-assertion\n")

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed() {
		t.Fatal("expected dangling depends-on to fail")
	}
	hasFailureContaining(t, result, "non-existent assertion")
}

// TestRun_DependsOnCycle_Fails covers checkCycles, the only non-trivial graph
// logic in the package: a 3-cycle must be flagged on every
// member (so the mistake is visible from whichever file a reader opens), not
// just one.
func TestRun_DependsOnCycle_Fails(t *testing.T) {
	specsDir := t.TempDir()
	writeSpec(t, specsDir, "my-spec", "")
	base := "parent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n"
	writeAssertion(t, specsDir, "my-spec", "a", "id: a\n"+base+"depends-on: b\n")
	writeAssertion(t, specsDir, "my-spec", "b", "id: b\n"+base+"depends-on: c\n")
	writeAssertion(t, specsDir, "my-spec", "c", "id: c\n"+base+"depends-on: a\n")

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cycleFailures := 0
	for _, f := range result.Failures {
		if strings.Contains(f.Message, "depends-on cycle") {
			cycleFailures++
		}
	}
	if cycleFailures != 3 {
		t.Errorf("expected all 3 cycle members flagged, got %d cycle failures in: %v", cycleFailures, result.Failures)
	}
}

func TestRun_NoSpecsDir_PassesEmpty(t *testing.T) {
	specsDir := filepath.Join(t.TempDir(), "does-not-exist")

	result, err := Run(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed() {
		t.Fatalf("expected no specs/ dir to pass trivially, got: %v", result.Failures)
	}
}
