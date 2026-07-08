package crossbranch

import (
	"os/exec"
	"strings"
	"testing"
)

// gitInRepo runs an arbitrary git command in dir, failing the test on error.
// Used only by the test harness to set up and inspect a scratch repo — the
// package itself never runs git this way.
func gitInRepo(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepoWithDirtyTree builds a temp git repo with a committed file and an
// uncommitted modification, then chdirs into it for the duration of the test.
// The dirty working tree exercises the "guarantee holds even with uncommitted
// changes" success criterion.
func newRepoWithDirtyTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	gitInRepo(t, dir, "init", "-q")
	gitInRepo(t, dir, "config", "user.email", "test@example.com")
	gitInRepo(t, dir, "config", "user.name", "Test")
	gitInRepo(t, dir, "config", "commit.gpgsign", "false")

	writeFile(t, dir, "spec.md", "original contents\n")
	gitInRepo(t, dir, "add", "spec.md")
	gitInRepo(t, dir, "commit", "-q", "-m", "initial")

	// Leave the working tree dirty.
	writeFile(t, dir, "spec.md", "edited but not committed\n")
	writeFile(t, dir, "untracked.md", "brand new\n")

	chdir(t, dir)
	return dir
}

func writeFile(t *testing.T, dir, name, contents string) {
	t.Helper()
	cmd := exec.Command("sh", "-c", "cat > "+name)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(contents)
	if err := cmd.Run(); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// snapshot captures the observable repo state that read-only operations must
// not change: HEAD commit, current branch, and the full porcelain status
// (which covers working tree + index + untracked files).
type snapshot struct {
	head   string
	branch string
	status string
}

func takeSnapshot(t *testing.T, dir string) snapshot {
	t.Helper()
	return snapshot{
		head:   gitInRepo(t, dir, "rev-parse", "HEAD"),
		branch: gitInRepo(t, dir, "rev-parse", "--abbrev-ref", "HEAD"),
		status: gitInRepo(t, dir, "status", "--porcelain"),
	}
}

func TestReadOnlyGuaranteeLeavesRepoUnchanged(t *testing.T) {
	dir := newRepoWithDirtyTree(t)

	before := takeSnapshot(t, dir)

	// Representative read-only operations the crossbranch flow relies on, all
	// routed through the single chokepoint. "branch" is intentionally absent:
	// it was removed from the allowlist because for-each-ref covers all
	// listing needs with no mutating flag surface.
	mustRun(t, "--version")
	mustRun(t, "rev-parse", "HEAD")
	mustRun(t, "for-each-ref", "--format=%(refname)")
	mustRun(t, "diff", "HEAD")
	mustRun(t, "show", "HEAD:spec.md")
	mustRun(t, "ls-tree", "HEAD")
	mustRun(t, "merge-base", "HEAD", "HEAD")

	after := takeSnapshot(t, dir)

	if before != after {
		t.Fatalf("read-only operations mutated the repo:\nbefore: %+v\nafter:  %+v", before, after)
	}
}

func TestReadOnlyGuaranteeRejectsMutatingSubcommands(t *testing.T) {
	dir := newRepoWithDirtyTree(t)
	before := takeSnapshot(t, dir)

	mutating := [][]string{
		{"commit", "-m", "should not run", "--allow-empty"},
		{"checkout", "-b", "newbranch"},
		{"switch", "-c", "newbranch"},
		{"merge", "HEAD"},
		{"reset", "--hard"},
		{"stash"},
		{"add", "."},
		{"branch", "-D", "main"},
		{"branch", "-m", "renamed"},
		{"branch", "-f", "main", "HEAD"},
		{"branch", "newbranch"},
		{"branch", "--set-upstream-to", "main"}, // separate-argument upstream form
		{"branch", "--set-upstream-to=main"},    // equals form must also be rejected
	}

	for _, args := range mutating {
		out, err := Run(args...)
		if err == nil {
			t.Fatalf("expected Run(%v) to be rejected, but it returned %q", args, out)
		}
	}

	// Allowing the rejected commands to slip through would change the repo;
	// confirm nothing moved.
	after := takeSnapshot(t, dir)
	if before != after {
		t.Fatalf("a rejected command appears to have executed:\nbefore: %+v\nafter:  %+v", before, after)
	}
}

func mustRun(t *testing.T, args ...string) {
	t.Helper()
	if _, err := Run(args...); err != nil {
		t.Fatalf("Run(%v) unexpectedly failed: %v", args, err)
	}
}

// TestAppendQuotePathConfigChains verifies the core.quotePath=false config is
// appended after any GIT_CONFIG entries the caller already injected, bumping the
// count instead of clobbering it, and that exactly one GIT_CONFIG_COUNT= entry
// remains in the result (no stale duplicate that would shadow the updated count
// on Linux/glibc).
func TestAppendQuotePathConfigChains(t *testing.T) {
	t.Run("no pre-existing config uses index 0", func(t *testing.T) {
		out := appendQuotePathConfig([]string{"PATH=/usr/bin"})
		want := []string{"GIT_CONFIG_KEY_0=core.quotePath", "GIT_CONFIG_VALUE_0=false", "GIT_CONFIG_COUNT=1"}
		assertTail(t, out, want)
		assertExactlyOne(t, out, "GIT_CONFIG_COUNT=")
	})

	t.Run("existing count is preserved and bumped", func(t *testing.T) {
		env := []string{
			"GIT_CONFIG_COUNT=2",
			"GIT_CONFIG_KEY_0=user.name", "GIT_CONFIG_VALUE_0=Alice",
			"GIT_CONFIG_KEY_1=core.pager", "GIT_CONFIG_VALUE_1=cat",
		}
		out := appendQuotePathConfig(env)
		want := []string{"GIT_CONFIG_KEY_2=core.quotePath", "GIT_CONFIG_VALUE_2=false", "GIT_CONFIG_COUNT=3"}
		assertTail(t, out, want)
		// The caller's two entries must still be present.
		if !containsLine(out, "GIT_CONFIG_KEY_0=user.name") || !containsLine(out, "GIT_CONFIG_KEY_1=core.pager") {
			t.Errorf("pre-existing GIT_CONFIG entries were dropped: %v", out)
		}
		assertExactlyOne(t, out, "GIT_CONFIG_COUNT=")
	})

	t.Run("stale GIT_CONFIG_COUNT is removed so no duplicate shadows the new count", func(t *testing.T) {
		// Simulate the scenario that triggers the Linux/glibc bug: env already
		// contains a GIT_CONFIG_COUNT (e.g. from a parent process) before we
		// inject ours. The result must contain exactly one GIT_CONFIG_COUNT=
		// so that getenv returns the correct updated value.
		env := []string{
			"GIT_CONFIG_COUNT=1",
			"GIT_CONFIG_KEY_0=user.name", "GIT_CONFIG_VALUE_0=Alice",
		}
		out := appendQuotePathConfig(env)
		assertExactlyOne(t, out, "GIT_CONFIG_COUNT=")
		if !containsLine(out, "GIT_CONFIG_COUNT=2") {
			t.Errorf("expected GIT_CONFIG_COUNT=2 in output, got: %v", out)
		}
	})

	t.Run("unparseable count falls back to 0", func(t *testing.T) {
		if got := existingConfigCount([]string{"GIT_CONFIG_COUNT=notanumber"}); got != 0 {
			t.Errorf("existingConfigCount(garbage) = %d, want 0", got)
		}
	})
}

func assertTail(t *testing.T, got, wantTail []string) {
	t.Helper()
	if len(got) < len(wantTail) {
		t.Fatalf("output too short: %v", got)
	}
	tail := got[len(got)-len(wantTail):]
	for i := range wantTail {
		if tail[i] != wantTail[i] {
			t.Fatalf("tail[%d] = %q, want %q (full: %v)", i, tail[i], wantTail[i], got)
		}
	}
}

func containsLine(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// assertExactlyOne asserts that exactly one entry in ss has the given prefix.
func assertExactlyOne(t *testing.T, ss []string, prefix string) {
	t.Helper()
	count := 0
	for _, s := range ss {
		if strings.HasPrefix(s, prefix) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one entry with prefix %q, got %d in %v", prefix, count, ss)
	}
}
