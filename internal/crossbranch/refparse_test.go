package crossbranch

import (
	"errors"
	"testing"
)

// newRepoWithSpecAtRef builds a temp git repo with a spec file committed on the
// default branch and chdirs into it. It returns the resolved name of the
// initial branch so tests can address it as a ref. The spec file holds valid
// frontmatter so it can be round-tripped through the existing parser.
func newRepoWithSpecAtRef(t *testing.T) (dir, branch string) {
	t.Helper()
	dir = t.TempDir()

	gitInRepo(t, dir, "init", "-q")
	gitInRepo(t, dir, "config", "user.email", "test@example.com")
	gitInRepo(t, dir, "config", "user.name", "Test")
	gitInRepo(t, dir, "config", "commit.gpgsign", "false")

	writeFile(t, dir, "spec.md",
		"---\nid: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: 2\nstatus: done\n---\n# My Spec Title\n\nBody.\n")
	gitInRepo(t, dir, "add", "spec.md")
	gitInRepo(t, dir, "commit", "-q", "-m", "add spec")

	chdir(t, dir)
	branch = gitInRepo(t, dir, "rev-parse", "--abbrev-ref", "HEAD")
	return dir, branch
}

func TestFileAtRef_ReturnsContentForPresentPath(t *testing.T) {
	_, branch := newRepoWithSpecAtRef(t)

	content, err := FileAtRef(branch, "spec.md")
	if err != nil {
		t.Fatalf("FileAtRef present path: %v", err)
	}
	if content == "" || !contains(content, "id: my-spec") {
		t.Fatalf("FileAtRef returned unexpected content: %q", content)
	}
}

func TestFileAtRef_SignalsAbsenceForMissingPath(t *testing.T) {
	_, branch := newRepoWithSpecAtRef(t)

	_, err := FileAtRef(branch, "nope.md")
	if !errors.Is(err, ErrFileAbsent) {
		t.Fatalf("missing path: got err %v, want ErrFileAbsent", err)
	}
}

func TestFileAtRef_RealErrorIsNotAbsence(t *testing.T) {
	// An unknown ref is a genuine failure and must NOT be reported as absence,
	// so callers don't mistake a broken ref for "spec not on that branch".
	newRepoWithSpecAtRef(t)

	_, err := FileAtRef("no-such-ref", "spec.md")
	if err == nil {
		t.Fatal("unknown ref: expected error, got nil")
	}
	if errors.Is(err, ErrFileAbsent) {
		t.Fatalf("unknown ref must not be reported as absence: %v", err)
	}
}

func TestSpecAtRef_ParsesFileReadFromRef(t *testing.T) {
	_, branch := newRepoWithSpecAtRef(t)

	spec, err := SpecAtRef(branch, "spec.md")
	if err != nil {
		t.Fatalf("SpecAtRef: %v", err)
	}
	if spec.ID != "my-spec" {
		t.Errorf("ID = %q, want %q", spec.ID, "my-spec")
	}
	if spec.Status != "done" {
		t.Errorf("Status = %q, want %q", spec.Status, "done")
	}
	if spec.Title != "My Spec Title" {
		t.Errorf("Title = %q, want %q", spec.Title, "My Spec Title")
	}
}

func TestSpecAtRef_AbsentFilePropagatesSentinel(t *testing.T) {
	_, branch := newRepoWithSpecAtRef(t)

	_, err := SpecAtRef(branch, "missing.md")
	if !errors.Is(err, ErrFileAbsent) {
		t.Fatalf("SpecAtRef absent: got %v, want ErrFileAbsent", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
