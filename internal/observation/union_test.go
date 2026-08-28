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

	// Deleting a branch forgets its observation: nothing claims the finding
	// any more, and a re-scan may legitimately file it again.
	git(t, dir, "branch", "-q", "-D", "observer/finding-a")
	u2, err := LoadUnion()
	if err != nil {
		t.Fatalf("LoadUnion after delete: %v", err)
	}
	if c := u2.FindCovering(TypeCodeSpecMisalignment, "finding-a"); c != nil {
		t.Fatalf("a deleted branch must be forgotten; still claimed at %q", c.Ref)
	}
	if u2.FindCovering(TypeCodeSpecMisalignment, "finding-b") == nil {
		t.Fatal("finding-b is still on its own branch and must still claim its finding")
	}
}

// Every observer branch is cut from origin/main, so every branch carries a
// copy of every observation already merged. A copy is not a claim: only the
// branch named after an observation speaks for it, and only until the slug
// reaches main.
func TestOnlyTheOwningBranchCovers(t *testing.T) {
	dir := newRepo(t)

	// One finding merged to main, resolved long ago.
	if err := os.MkdirAll(filepath.Join(dir, "observations"), 0o755); err != nil {
		t.Fatal(err)
	}
	resolved := strings.Replace(validObs(map[string]string{"slug": "old-drift"}), "status: open", "status: resolved", 1)
	if err := os.WriteFile(filepath.Join(dir, "observations/old-drift.md"), []byte(resolved), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "merge old-drift")

	// An unrelated finding filed afterwards inherits old-drift.md.
	commitObservation(t, dir, "observer/unrelated", "observations/unrelated.md",
		validObs(map[string]string{"slug": "unrelated"}))

	u, err := LoadUnion()
	if err != nil {
		t.Fatalf("LoadUnion: %v", err)
	}
	if c := u.FindCovering(TypeCodeSpecMisalignment, "old-drift"); c != nil {
		t.Fatalf("an inherited copy must not claim the finding; claimed at %q", c.Ref)
	}
	claim := u.FindCovering(TypeCodeSpecMisalignment, "unrelated")
	if claim == nil || claim.Ref != "refs/heads/observer/unrelated" {
		t.Fatalf("the live finding must be claimed from its own branch, got %+v", claim)
	}

	// A branch merged but never deleted carries the observation at its own
	// name. Main wins: the finding is resolved while the branch survives.
	git(t, dir, "branch", "-q", "observer/old-drift", "main")
	u2, err := LoadUnion()
	if err != nil {
		t.Fatalf("LoadUnion after restoring the merged branch: %v", err)
	}
	if c := u2.FindCovering(TypeCodeSpecMisalignment, "old-drift"); c != nil {
		t.Fatalf("a slug on main must not claim, even from its own branch; claimed at %q", c.Ref)
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
