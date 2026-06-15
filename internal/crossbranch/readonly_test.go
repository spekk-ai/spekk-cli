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

	t.Chdir(dir)
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
	// routed through the single chokepoint.
	mustRun(t, "--version")
	mustRun(t, "rev-parse", "HEAD")
	mustRun(t, "for-each-ref", "--format=%(refname)")
	mustRun(t, "branch", "--list")
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
	_ = dir
}

func mustRun(t *testing.T, args ...string) {
	t.Helper()
	if _, err := Run(args...); err != nil {
		t.Fatalf("Run(%v) unexpectedly failed: %v", args, err)
	}
}
