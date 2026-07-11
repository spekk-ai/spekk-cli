package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/config"
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

	// Skill name with traversal segments — would resolve outside the agent dir.
	_, err := Uninstall(cwd, "/home/u", ScopeLocal, "coach", "../../../outside")
	if err == nil {
		t.Fatal("expected error when target escapes scope directory")
	}
	// Must be refused for a path-safety reason, NOT a "not installed" error —
	// "not installed" would pass even if the guard was missing, since the
	// traversed path happens to be empty in a tempdir. The name is rejected at
	// validation (invalid skill name) before the scope guard is even reached;
	// either path-safety rejection is acceptable, but "not installed" is not.
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "scope") && !strings.Contains(msg, "invalid skill name") {
		t.Errorf("error should be a path-safety refusal (scope or invalid skill name), got: %s", err)
	}

	// The outside file must still exist.
	if _, statErr := os.Stat(outside); statErr != nil {
		t.Errorf("uninstall touched a file outside the scope dir: %v", statErr)
	}
}

func TestUninstall_GlobalScopeUsesConfigDir(t *testing.T) {
	xdgBase := t.TempDir()
	config.ResetCacheForTest(t)
	t.Setenv("XDG_CONFIG_HOME", xdgBase)

	dest, _ := Destination("/some/cwd", "", ScopeGlobal, "coach", "meeting-notes")
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(dest, []byte("body"), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	removed, err := Uninstall("/some/cwd", "", ScopeGlobal, "coach", "meeting-notes")
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	wantPrefix := filepath.Join(xdgBase, "spekk", "skills", "coach")
	if !strings.HasPrefix(removed, wantPrefix) {
		t.Errorf("global scope: removed path %q should be under %q", removed, wantPrefix)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("file still exists after global uninstall: %v", err)
	}
}

func TestParseUninstallArgs_DefaultLocalScope(t *testing.T) {
	opts, err := ParseUninstallArgs([]string{"coach", "meeting-notes"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Agent != "coach" || opts.Skill != "meeting-notes" {
		t.Errorf("positionals: agent=%q skill=%q", opts.Agent, opts.Skill)
	}
	if opts.Scope != ScopeLocal {
		t.Errorf("scope: want local, got %s", opts.Scope)
	}
}

func TestParseUninstallArgs_GlobalScope(t *testing.T) {
	opts, err := ParseUninstallArgs([]string{"coach", "meeting-notes", "--global"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Scope != ScopeGlobal {
		t.Errorf("scope: want global, got %s", opts.Scope)
	}
}

func TestParseUninstallArgs_GlobalAndLocalConflict(t *testing.T) {
	_, err := ParseUninstallArgs([]string{"coach", "foo", "--global", "--local"})
	if err == nil {
		t.Fatal("expected error for --global + --local")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "mutually exclusive") {
		t.Errorf("error should explain mutual exclusion, got: %s", err)
	}
}

func TestParseUninstallArgs_MissingSkill(t *testing.T) {
	_, err := ParseUninstallArgs([]string{"coach"})
	if err == nil {
		t.Fatal("expected error when <skill> is omitted")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "skill") {
		t.Errorf("error should mention missing skill, got: %s", err)
	}
}

func TestParseUninstallArgs_UnknownAgent(t *testing.T) {
	_, err := ParseUninstallArgs([]string{"bogus", "foo"})
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	for _, valid := range ValidAgents {
		if !strings.Contains(err.Error(), valid) {
			t.Errorf("error should list valid agent %q, got: %s", valid, err)
		}
	}
}
