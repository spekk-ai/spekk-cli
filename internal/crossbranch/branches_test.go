package crossbranch

import (
	"os"
	"os/exec"
	"sort"
	"testing"
)

// chdir changes the process working directory to dir for the duration of the
// test, restoring the previous cwd via t.Cleanup. Used instead of t.Chdir, which
// is a go1.24 API, to keep the module's declared go1.23 toolchain floor.
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

// git runs a raw git command in dir, failing the test on error. Tests need
// write commands (init, commit, update-ref) to build fixtures, so they bypass
// the read-only Run chokepoint and call git directly — Run itself is exercised
// indirectly through DiscoverBranches.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// newRepo creates a temp git repo with one commit on branch "main" and chdirs
// into it so DiscoverBranches (which shells out in the process cwd) operates on
// it. Returns the commit sha that all fixture refs can point at.
func newRepo(t *testing.T) (dir, head string) {
	t.Helper()
	dir = t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	head = trim(git(t, dir, "rev-parse", "HEAD"))
	chdir(t, dir)
	return dir, head
}

func trim(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

func names(bs []Branch) []string {
	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = b.Name
	}
	sort.Strings(out)
	return out
}

func eq(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v, want %v", label, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s: got %v, want %v", label, got, want)
		}
	}
}

// TestDiscoverBranches covers the assertion's behavior in one fixture:
// current branch excluded, local+remote union deduped, origin/HEAD skipped,
// and the filter glob applied. HEAD is on "main".
func TestDiscoverBranches(t *testing.T) {
	dir, head := newRepo(t)

	// Local branches.
	git(t, dir, "branch", "feature-a")
	git(t, dir, "branch", "feature-b")
	git(t, dir, "branch", "shared") // also exists on origin -> must dedup

	// Remote-tracking refs, created directly so no network is needed.
	git(t, dir, "update-ref", "refs/remotes/origin/shared", head) // dup of local
	git(t, dir, "update-ref", "refs/remotes/origin/remote-only", head)
	git(t, dir, "update-ref", "refs/remotes/origin/HEAD", head) // must be skipped
	git(t, dir, "update-ref", "refs/remotes/origin/main", head) // dup of current

	t.Run("union deduped, current and origin/HEAD excluded", func(t *testing.T) {
		bs, err := DiscoverBranches("")
		if err != nil {
			t.Fatal(err)
		}
		// Expected: locals feature-a, feature-b, shared + remote-only.
		// Excluded: main (current, both local and origin/main), origin/HEAD,
		// and origin/shared collapses into local shared.
		eq(t, "names", names(bs), []string{"feature-a", "feature-b", "remote-only", "shared"})
	})

	t.Run("shared collapses to the local head", func(t *testing.T) {
		bs, err := DiscoverBranches("shared")
		if err != nil {
			t.Fatal(err)
		}
		if len(bs) != 1 {
			t.Fatalf("expected 1 branch for filter shared, got %v", names(bs))
		}
		b := bs[0]
		if b.Remote {
			t.Errorf("shared should resolve to the local head, got Remote=true")
		}
		if b.Ref != "refs/heads/shared" {
			t.Errorf("shared Ref = %q, want refs/heads/shared", b.Ref)
		}
		if b.Rev != "refs/heads/shared" {
			t.Errorf("shared Rev = %q, want refs/heads/shared (merge-tree target)", b.Rev)
		}
	})

	t.Run("remote-only branch is flagged remote", func(t *testing.T) {
		bs, err := DiscoverBranches("remote-only")
		if err != nil {
			t.Fatal(err)
		}
		if len(bs) != 1 || !bs[0].Remote || bs[0].Ref != "refs/remotes/origin/remote-only" {
			t.Fatalf("remote-only not resolved from remote ref: %+v", bs)
		}
	})

	t.Run("filter glob includes and excludes by name", func(t *testing.T) {
		bs, err := DiscoverBranches("feature-*")
		if err != nil {
			t.Fatal(err)
		}
		eq(t, "glob feature-*", names(bs), []string{"feature-a", "feature-b"})
	})

	t.Run("invalid glob errors", func(t *testing.T) {
		if _, err := DiscoverBranches("[invalid"); err == nil {
			t.Fatal("expected error for malformed glob pattern")
		}
	})
}

// TestDiscoverBranchesEmpty: a repo whose only branch is the current one yields
// an empty comparison set (graceful no-op), not an error.
func TestDiscoverBranchesEmpty(t *testing.T) {
	newRepo(t) // only "main" exists, and it's current
	bs, err := DiscoverBranches("")
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) != 0 {
		t.Fatalf("expected empty set with no other branches, got %v", names(bs))
	}
}

// TestDiscoverBranchesDetachedHead: with HEAD detached there is no "ours" to
// exclude, so every branch participates and nothing crashes.
func TestDiscoverBranchesDetachedHead(t *testing.T) {
	dir, head := newRepo(t)
	git(t, dir, "branch", "other")
	git(t, dir, "checkout", "-q", "--detach", head)

	bs, err := DiscoverBranches("")
	if err != nil {
		t.Fatal(err)
	}
	// main is no longer "current" since HEAD is detached, so both heads show.
	eq(t, "detached", names(bs), []string{"main", "other"})
}
