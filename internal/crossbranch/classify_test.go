package crossbranch

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSpec writes a file at repo-relative path (creating parent dirs) in dir.
func writeSpec(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// removeSpec deletes a repo-relative file from dir.
func removeSpec(t *testing.T, dir, relPath string) {
	t.Helper()
	if err := os.Remove(filepath.Join(dir, filepath.FromSlash(relPath))); err != nil {
		t.Fatal(err)
	}
}

// assertionMD builds minimal valid assertion file content with the given id and
// status, parseable by the existing parser.
func assertionMD(id, status, body string) string {
	return "---\n" +
		"id: " + id + "\n" +
		"parent: demo\n" +
		"created: 2026-01-01T00:00:00Z\n" +
		"priority: 1\n" +
		"status: " + status + "\n" +
		"---\n\n# " + id + "\n\n" + body + "\n"
}

// find returns the FileState for (path, branch) or fails the test.
func find(t *testing.T, states []FileState, path, branch string) FileState {
	t.Helper()
	for _, s := range states {
		if s.Path == path && s.Branch == branch {
			return s
		}
	}
	t.Fatalf("no FileState for %s on %s; got %+v", path, branch, states)
	return FileState{}
}

// TestClassifyStates exercises all four states plus status drift in one fixture
// repo. "main" is ours; a single "other" branch diverges from a common base.
func TestClassifyStates(t *testing.T) {
	dir, _ := newRepo(t)

	// Base commit on main: a spec parent and three assertion files that the
	// branch will modify / delete, plus one assertion that drifts in status.
	const (
		specFile  = "specs/demo/demo.md"
		modFile   = "specs/demo/assertions/clean-mod.md"
		delFile   = "specs/demo/assertions/to-delete.md"
		conflFile = "specs/demo/assertions/conflicted.md"
		driftFile = "specs/demo/assertions/drifts.md"
		addFile   = "specs/demo/assertions/foreign.md" // added only on branch
		unchanged = "specs/demo/assertions/stable.md"
	)

	writeSpec(t, dir, specFile, "---\nid: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n\n# demo\n")
	writeSpec(t, dir, modFile, assertionMD("clean-mod", "not_started", "original body"))
	writeSpec(t, dir, delFile, assertionMD("to-delete", "not_started", "doomed"))
	writeSpec(t, dir, conflFile, assertionMD("conflicted", "not_started", "shared region line A\nshared region line B\nshared region line C"))
	writeSpec(t, dir, driftFile, assertionMD("drifts", "not_started", "drift body"))
	writeSpec(t, dir, unchanged, assertionMD("stable", "done", "never changes"))
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "base")

	// Branch diverges.
	git(t, dir, "checkout", "-q", "-b", "other")
	writeSpec(t, dir, addFile, assertionMD("foreign", "draft", "only on branch"))             // incoming add
	writeSpec(t, dir, modFile, assertionMD("clean-mod", "not_started", "branch-edited body")) // clean mod (body only)
	removeSpec(t, dir, delFile)                                                               // incoming deletion
	writeSpec(t, dir, driftFile, assertionMD("drifts", "done", "drift body"))                 // status drift only
	writeSpec(t, dir, conflFile, assertionMD("conflicted", "not_started", "shared region line A\nBRANCH CHANGE\nshared region line C"))
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "branch work")

	// Back to ours and edit the conflicted file's same region differently.
	git(t, dir, "checkout", "-q", "main")
	writeSpec(t, dir, conflFile, assertionMD("conflicted", "not_started", "shared region line A\nOURS CHANGE\nshared region line C"))
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "ours work")

	states, err := Classify("")
	if err != nil {
		t.Fatal(err)
	}

	// Incoming addition.
	if s := find(t, states, addFile, "other"); s.State != StateIncomingAdd {
		t.Errorf("%s: State = %q, want %q", addFile, s.State, StateIncomingAdd)
	}

	// Clean incoming modification (no status change -> no drift detail).
	if s := find(t, states, modFile, "other"); s.State != StateIncomingMod {
		t.Errorf("%s: State = %q, want %q", modFile, s.State, StateIncomingMod)
	}

	// Incoming deletion.
	if s := find(t, states, delFile, "other"); s.State != StateIncomingDel {
		t.Errorf("%s: State = %q, want %q", delFile, s.State, StateIncomingDel)
	}

	// Real conflict (same region edited on both sides). merge-tree must report
	// it; on a modern git Degraded is false.
	confl := find(t, states, conflFile, "other")
	if confl.State != StateConflict {
		t.Errorf("%s: State = %q, want %q", conflFile, confl.State, StateConflict)
	}
	if ok, _ := SupportsMergeTree(); ok && confl.Degraded {
		t.Errorf("%s: Degraded = true on git that supports merge-tree", conflFile)
	}

	// Status drift on a clean modification.
	drift := find(t, states, driftFile, "other")
	if drift.State != StateIncomingMod {
		t.Errorf("%s: State = %q, want %q", driftFile, drift.State, StateIncomingMod)
	}
	if drift.OldStatus != "not_started" || drift.NewStatus != "done" {
		t.Errorf("%s: drift = (%q -> %q), want (not_started -> done)", driftFile, drift.OldStatus, drift.NewStatus)
	}

	// Unchanged file contributes nothing.
	for _, s := range states {
		if s.Path == unchanged {
			t.Errorf("unchanged file %s should contribute no FileState, got %+v", unchanged, s)
		}
	}
}

// TestClassifyEmpty: no other branches -> empty result, no error.
func TestClassifyEmpty(t *testing.T) {
	newRepo(t)
	states, err := Classify("")
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 0 {
		t.Fatalf("expected no contributions with no comparison branches, got %+v", states)
	}
}

// TestClassifySkipsUnrelatedHistory: a branch sharing no common ancestor with
// HEAD (orphan history) has no three-way merge-base. It must be skipped
// (contribute nothing) without failing the whole classification.
func TestClassifySkipsUnrelatedHistory(t *testing.T) {
	dir, _ := newRepo(t)

	// A spec on main (ours).
	writeSpec(t, dir, "specs/demo/demo.md", "---\nid: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n\n# demo\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "spec on main")

	// An orphan branch: a completely unrelated history that adds its own spec.
	git(t, dir, "checkout", "-q", "--orphan", "orphan")
	git(t, dir, "rm", "-rf", ".") // clear main's tracked files from index + worktree
	writeSpec(t, dir, "specs/foreign/foreign.md", "---\nid: foreign\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n\n# foreign\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "orphan root")
	git(t, dir, "checkout", "-q", "main")

	states, err := Classify("")
	if err != nil {
		t.Fatalf("Classify must not fail on an unrelated-history branch: %v", err)
	}
	for _, s := range states {
		if s.Branch == "orphan" {
			t.Errorf("orphan branch (no merge-base) should contribute nothing, got %+v", s)
		}
	}
}

// TestClassifyBranchDegradedMode exercises the git<2.38 degraded path that the
// real SupportsMergeTree() can never reach on a modern test host. It drives
// classifyBranch directly with mergeTreeOK=false and contrasts it against the
// supported path on the same fixture: in supported mode a both-changed file that
// merges cleanly is incoming_mod and only the overlapping edit is a confirmed
// conflict; in degraded mode every both-changed file becomes an *unconfirmed*
// (Degraded) conflict, with no silent drops.
func TestClassifyBranchDegradedMode(t *testing.T) {
	dir, _ := newRepo(t)

	const (
		conflFile = "specs/demo/assertions/conflicted.md"
		cleanBoth = "specs/demo/assertions/clean-both.md"
	)

	// Base on main.
	writeSpec(t, dir, conflFile, assertionMD("conflicted", "not_started", "L1\nL2\nL3"))
	writeSpec(t, dir, cleanBoth, assertionMD("clean-both", "not_started", "A1\nA2\nA3\nA4\nA5\nA6\nA7"))
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "base")

	// "other" edits both files.
	git(t, dir, "checkout", "-q", "-b", "other")
	writeSpec(t, dir, conflFile, assertionMD("conflicted", "not_started", "L1\nTHEIRS\nL3"))
	writeSpec(t, dir, cleanBoth, assertionMD("clean-both", "not_started", "A1\nA2\nA3\nA4\nA5\nA6\nTHEIRS7"))
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "other edits")

	// Ours (main) edits the conflicted file's same region, and the clean file's
	// far region (so it merges cleanly).
	git(t, dir, "checkout", "-q", "main")
	writeSpec(t, dir, conflFile, assertionMD("conflicted", "not_started", "L1\nOURS\nL3"))
	writeSpec(t, dir, cleanBoth, assertionMD("clean-both", "not_started", "OURS1\nA2\nA3\nA4\nA5\nA6\nA7"))
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "our edits")

	b := Branch{Name: "other", Rev: "refs/heads/other"}

	// Supported mode: overlapping edit is a confirmed conflict (Degraded=false);
	// the non-overlapping both-edit merges cleanly -> incoming_mod.
	supported, err := classifyBranch(b, true)
	if err != nil {
		t.Fatal(err)
	}
	if s := find(t, supported, conflFile, "other"); s.State != StateConflict || s.Degraded {
		t.Errorf("supported conflFile: got %+v, want StateConflict with Degraded=false", s)
	}
	if s := find(t, supported, cleanBoth, "other"); s.State != StateIncomingMod {
		t.Errorf("supported cleanBoth: got %+v, want StateIncomingMod", s)
	}

	// Degraded mode: both both-changed files become unconfirmed conflicts.
	degraded, err := classifyBranch(b, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{conflFile, cleanBoth} {
		s := find(t, degraded, f, "other")
		if s.State != StateConflict || !s.Degraded {
			t.Errorf("degraded %s: got %+v, want StateConflict with Degraded=true", f, s)
		}
	}
}

// TestClassifyFromSubdirectory: cross-branch git must anchor at the repo root,
// so classification works even when spekk is invoked from a subdirectory. Before
// the cmd.Dir anchor the "-- specs" pathspec resolved relative to cwd and matched
// nothing, silently yielding an empty overlay.
func TestClassifyFromSubdirectory(t *testing.T) {
	dir, _ := newRepo(t)

	writeSpec(t, dir, "specs/demo/demo.md", "---\nid: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n\n# demo\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "base")

	git(t, dir, "checkout", "-q", "-b", "other")
	writeSpec(t, dir, "specs/foreign/foreign.md", "---\nid: foreign\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n\n# foreign\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "foreign")
	git(t, dir, "checkout", "-q", "main")

	// Invoke from a nested subdirectory of the repo.
	chdir(t, filepath.Join(dir, "specs", "demo"))

	states, err := Classify("")
	if err != nil {
		t.Fatal(err)
	}
	if s := find(t, states, "specs/foreign/foreign.md", "other"); s.State != StateIncomingAdd {
		t.Errorf("from subdir: got %+v, want incoming_add", s)
	}
}

// TestParseMergeTreeConflicts validates conflict-report parsing against the
// real --name-only output shape for both clean and conflicted merges.
func TestParseMergeTreeConflicts(t *testing.T) {
	clean := "1a764d07c2241e5bd8e9046ee16194747bac7030\n"
	if got := parseMergeTreeConflicts(clean); len(got) != 0 {
		t.Errorf("clean merge: got conflicts %v, want none", got)
	}

	conflicted := "bb9da2946f93d5a977237a1be6489e184bf4a787\n" +
		"specs/demo/assertions/conflicted.md\n" +
		"\n" +
		"Auto-merging specs/demo/assertions/conflicted.md\n" +
		"CONFLICT (content): Merge conflict in specs/demo/assertions/conflicted.md\n"
	got := parseMergeTreeConflicts(conflicted)
	if !got["specs/demo/assertions/conflicted.md"] {
		t.Errorf("conflicted merge: missing conflicted path, got %v", got)
	}
	if len(got) != 1 {
		t.Errorf("conflicted merge: got %d conflicts, want 1: %v", len(got), got)
	}
}
