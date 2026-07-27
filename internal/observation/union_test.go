package observation

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// chdir switches the process working directory for the test and restores it
// afterwards (same helper style as internal/crossbranch tests).
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepo creates a temp git repo on branch main with one commit and chdirs
// into it.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")
	git(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	chdir(t, dir)
	return dir
}

// commitObservation writes an observation file on a branch (created from
// main) and commits it, then returns to main.
func commitObservation(t *testing.T, dir, branch, file, content string) {
	t.Helper()
	git(t, dir, "checkout", "-q", "-b", branch, "main")
	full := filepath.Join(dir, file)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", file)
	git(t, dir, "commit", "-q", "-m", "add "+file)
	git(t, dir, "checkout", "-q", "main")
}

func TestLoadUnionReadsBranchesAndMain(t *testing.T) {
	dir := newRepo(t)

	commitObservation(t, dir, "observer/finding-a", "observations/finding-a.md",
		validObs(map[string]string{"slug": "finding-a"}))
	commitObservation(t, dir, "observer/finding-b", "observations/finding-b.md",
		validObs(map[string]string{"slug": "finding-b", "severity": "medium"}))
	// An invalid observation (evidence gate) must be skipped with a warning,
	// never silently indexed as valid.
	commitObservation(t, dir, "observer/no-evidence", "observations/no-evidence.md",
		validObs(map[string]string{"slug": "no-evidence", "affected": "@none"}))
	// A resolved observation merged to main.
	git(t, dir, "checkout", "-q", "main")
	if err := os.MkdirAll(filepath.Join(dir, "observations"), 0o755); err != nil {
		t.Fatal(err)
	}
	resolved := strings.Replace(validObs(map[string]string{"slug": "finding-c"}), "status: open", "status: resolved", 1)
	if err := os.WriteFile(filepath.Join(dir, "observations/finding-c.md"), []byte(resolved), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "observations/finding-c.md")
	git(t, dir, "commit", "-q", "-m", "merge finding-c")

	u, err := LoadUnion()
	if err != nil {
		t.Fatalf("LoadUnion: %v", err)
	}

	got := map[string]string{}
	for _, o := range u.Observations {
		got[o.Slug] = o.Ref
	}
	if len(u.Observations) != 3 {
		t.Fatalf("want 3 valid observations, got %d: %v", len(u.Observations), got)
	}
	if got["finding-a"] != "refs/heads/observer/finding-a" {
		t.Fatalf("finding-a ref: %q", got["finding-a"])
	}
	if !isMainRef(got["finding-c"]) {
		t.Fatalf("finding-c must come from main, got %q", got["finding-c"])
	}
	if len(u.Warnings) != 1 || !strings.Contains(u.Warnings[0], "no-evidence.md") {
		t.Fatalf("invalid file must produce a warning naming it: %v", u.Warnings)
	}

	// The working tree is never touched: no checkout of observer branches.
	if branch := git(t, dir, "rev-parse", "--abbrev-ref", "HEAD"); branch != "main" {
		t.Fatalf("working tree branch changed to %q", branch)
	}

	// Deleting a branch forgets its observation: the union no longer covers
	// the drift and a re-scan may legitimately re-flag it.
	git(t, dir, "branch", "-q", "-D", "observer/finding-a")
	u2, err := LoadUnion()
	if err != nil {
		t.Fatalf("LoadUnion after delete: %v", err)
	}
	if u2.FindCovering(TypeCodeSpecMisalignment, []string{"internal/parser/parser.go"}) == nil {
		t.Fatal("finding-b still covers the drift (same type, overlapping path)")
	}
	git(t, dir, "branch", "-q", "-D", "observer/finding-b")
	u3, err := LoadUnion()
	if err != nil {
		t.Fatalf("LoadUnion after second delete: %v", err)
	}
	if c := u3.FindCovering(TypeCodeSpecMisalignment, []string{"internal/parser/parser.go"}); c != nil {
		t.Fatalf("deleted branches must be forgotten; still covered by %q", c.Slug)
	}
}

func TestLoadUnionSeesRemoteTrackingRefs(t *testing.T) {
	// Build an origin with an observer branch, clone it, and confirm the
	// union sees the remote-tracking ref without any local branch.
	origin := t.TempDir()
	git(t, origin, "init", "-q", "-b", "main")
	git(t, origin, "config", "user.email", "test@example.com")
	git(t, origin, "config", "user.name", "Test")
	git(t, origin, "commit", "-q", "--allow-empty", "-m", "init")
	commitObservation(t, origin, "observer/remote-only", "observations/remote-only.md",
		validObs(map[string]string{"slug": "remote-only"}))

	clone := t.TempDir()
	git(t, clone, "clone", "-q", origin, ".")
	chdir(t, clone)

	u, err := LoadUnion()
	if err != nil {
		t.Fatalf("LoadUnion: %v", err)
	}
	var found *Observation
	for _, o := range u.Observations {
		if o.Slug == "remote-only" {
			found = o
		}
	}
	if found == nil {
		t.Fatalf("remote-tracking observer branch not in union: %+v", u.Observations)
	}
	if !strings.HasPrefix(found.Ref, "refs/remotes/") {
		t.Fatalf("expected a remote-tracking ref, got %q", found.Ref)
	}
}
