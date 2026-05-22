package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUninstall_RemovesFileWhenPresent(t *testing.T) {
	cwd := t.TempDir()
	dest, _ := Destination(cwd, "/home/u", ScopeLocal, "coach", "meeting-notes")
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(dest, []byte("body"), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	removed, err := Uninstall(cwd, "/home/u", ScopeLocal, "coach", "meeting-notes")
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if removed != dest {
		t.Errorf("returned path: got %q, want %q", removed, dest)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("file still exists after uninstall: %v", err)
	}
}

func TestUninstall_ErrorsWhenAbsent(t *testing.T) {
	cwd := t.TempDir()
	_, err := Uninstall(cwd, "/home/u", ScopeLocal, "coach", "missing-skill")
	if err == nil {
		t.Fatal("expected error when target file does not exist")
	}
	msg := err.Error()
	if !strings.Contains(strings.ToLower(msg), "not installed") {
		t.Errorf("error should say 'not installed', got: %s", msg)
	}
	wantPath := filepath.Join(cwd, ".spekk", "skills", "coach", "missing-skill.md")
	if !strings.Contains(msg, wantPath) {
		t.Errorf("error should name the path %q, got: %s", wantPath, msg)
	}
}

// Uninstall must only delete inside <scope>/.spekk/skills/<agent>/. A skill
// name with traversal segments must be refused without touching any file.
func TestUninstall_RefusesToTouchOutsideScopeDir(t *testing.T) {
	cwd := t.TempDir()
	// Seed an "outside" file that uninstall must NOT remove.
	outside := filepath.Join(cwd, "outside.md")
	if err := os.WriteFile(outside, []byte("do not delete"), 0644); err != nil {
		t.Fatalf("seed outside file: %v", err)
	}

	// Skill name with traversal segments — resolves outside the agent dir.
	_, err := Uninstall(cwd, "/home/u", ScopeLocal, "coach", "../../../outside")
	if err == nil {
		t.Fatal("expected error when target escapes scope directory")
	}
	// Must be the scope-guard rejection, NOT a "not installed" error — that
	// would pass even if the guard was missing, since the traversed path
	// happens to be empty in a tempdir.
	if !strings.Contains(strings.ToLower(err.Error()), "scope") {
		t.Errorf("error should mention scope-directory refusal, got: %s", err)
	}

	// The outside file must still exist.
	if _, statErr := os.Stat(outside); statErr != nil {
		t.Errorf("uninstall touched a file outside the scope dir: %v", statErr)
	}
}
