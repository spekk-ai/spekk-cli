package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveGlobalConfigDir_DefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := resolveGlobalConfigDir(home, io.Discard, strings.NewReader(""), false)
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(home, ".config", "spekk")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestResolveGlobalConfigDir_XDGOverride(t *testing.T) {
	home := t.TempDir()
	custom := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", custom)

	dir, err := resolveGlobalConfigDir(home, io.Discard, strings.NewReader(""), false)
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(custom, "spekk")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestDefaultDir_HonorsXDG(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", custom)
	if got := DefaultDir(); got != filepath.Join(custom, "spekk") {
		t.Errorf("expected XDG-based path, got %s", got)
	}
}

func TestMaybeMigrate_OldAbsent(t *testing.T) {
	base := t.TempDir()
	out := &strings.Builder{}
	err := maybeMigrate(filepath.Join(base, "old"), filepath.Join(base, "new"), out, strings.NewReader(""), true)
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() > 0 {
		t.Error("should produce no output when old dir is absent")
	}
}

func TestMaybeMigrate_BothExist(t *testing.T) {
	base := t.TempDir()
	oldDir := filepath.Join(base, "old")
	newDir := filepath.Join(base, "new")
	os.MkdirAll(oldDir, 0o755)
	os.MkdirAll(newDir, 0o755)

	out := &strings.Builder{}
	err := maybeMigrate(oldDir, newDir, out, strings.NewReader(""), true)
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() > 0 {
		t.Error("should produce no output when both dirs exist")
	}
	if _, err := os.Stat(oldDir); err != nil {
		t.Error("old dir should be left untouched when both exist")
	}
}

func TestMaybeMigrate_InteractiveMigratesOnEnter(t *testing.T) {
	base := t.TempDir()
	oldDir := filepath.Join(base, "old")
	newDir := filepath.Join(base, ".config", "spekk")
	os.MkdirAll(oldDir, 0o755)
	os.WriteFile(filepath.Join(oldDir, "file.txt"), []byte("data"), 0o644)

	out := &strings.Builder{}
	err := maybeMigrate(oldDir, newDir, out, strings.NewReader("\n"), true)
	if err != nil {
		t.Fatal(err)
	}
	if dirExists(oldDir) {
		t.Error("old dir should be gone after migration")
	}
	if !dirExists(newDir) {
		t.Error("new dir should exist after migration")
	}
	data, readErr := os.ReadFile(filepath.Join(newDir, "file.txt"))
	if readErr != nil || string(data) != "data" {
		t.Error("file should be present in new location after migration")
	}
	if !strings.Contains(out.String(), "Press Enter") {
		t.Error("interactive mode should prompt for Enter")
	}
	if !strings.Contains(out.String(), "Migrated") {
		t.Error("should print migration confirmation")
	}
}

func TestMaybeMigrate_NonInteractiveDoesNotPrompt(t *testing.T) {
	base := t.TempDir()
	oldDir := filepath.Join(base, "old")
	newDir := filepath.Join(base, "new")
	os.MkdirAll(oldDir, 0o755)
	os.WriteFile(filepath.Join(oldDir, "file.txt"), []byte("data"), 0o644)

	out := &strings.Builder{}
	// Empty reader: a blocking read here would return immediately with EOF,
	// but the point is non-interactive mode must never ask.
	err := maybeMigrate(oldDir, newDir, out, strings.NewReader(""), false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Press Enter") {
		t.Error("non-interactive mode must not prompt")
	}
	if !dirExists(newDir) {
		t.Error("migration should still happen non-interactively")
	}
	if !strings.Contains(out.String(), "Migrated") {
		t.Error("should still print migration notice")
	}
}

func TestMaybeMigrate_NestedFiles(t *testing.T) {
	base := t.TempDir()
	oldDir := filepath.Join(base, "old")
	newDir := filepath.Join(base, "new")
	subDir := filepath.Join(oldDir, "skills", "coach")
	os.MkdirAll(subDir, 0o755)
	os.WriteFile(filepath.Join(subDir, "skill.md"), []byte("# Skill"), 0o644)

	err := maybeMigrate(oldDir, newDir, io.Discard, strings.NewReader("\n"), true)
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := os.ReadFile(filepath.Join(newDir, "skills", "coach", "skill.md"))
	if readErr != nil || string(data) != "# Skill" {
		t.Error("nested files should be present in new location after migration")
	}
}

func TestMaybeMigrate_PromptText(t *testing.T) {
	base := t.TempDir()
	oldDir := filepath.Join(base, "old")
	newDir := filepath.Join(base, "new")
	os.MkdirAll(oldDir, 0o755)

	out := &strings.Builder{}
	maybeMigrate(oldDir, newDir, out, strings.NewReader("\n"), true)

	msg := out.String()
	if !strings.Contains(msg, oldDir) {
		t.Error("prompt should include old path")
	}
	if !strings.Contains(msg, newDir) {
		t.Error("prompt should include new path")
	}
}
