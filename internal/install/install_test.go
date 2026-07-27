package install

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// fakeSkillFS returns a minimal in-memory FS that satisfies the skill embed
// path, so tests don't need the real embedded binary asset.
func fakeSkillFS() fs.FS {
	return fstest.MapFS{
		skillEmbedPath: &fstest.MapFile{Data: []byte("# spekk-dev-loop\nfake skill content for tests")},
	}
}

// fakeSkillContentWithFrontmatter mirrors the real embedded skill's shape
// (YAML frontmatter, blank line, then the body) so strip-specific tests
// actually exercise stripFrontmatter instead of a no-op.
const fakeSkillContentWithFrontmatter = "---\nname: spekk-dev-loop\ndescription: \"fake\"\n---\n\n# Spekk Dev Loop\nfake skill content for tests\n"

// fakeSkillFSWithFrontmatter returns an in-memory FS whose skill content has
// a real leading frontmatter block, for command/prompt targets that strip it.
func fakeSkillFSWithFrontmatter() fs.FS {
	return fstest.MapFS{
		skillEmbedPath: &fstest.MapFile{Data: []byte(fakeSkillContentWithFrontmatter)},
	}
}

// TestStripFrontmatter covers the shared helper directly: it must remove a
// leading YAML frontmatter block through the closing "---" and the single
// blank line after it, and must leave content without a leading "---\n"
// untouched.
func TestStripFrontmatter(t *testing.T) {
	got := stripFrontmatter([]byte(fakeSkillContentWithFrontmatter))
	want := "# Spekk Dev Loop\nfake skill content for tests\n"
	if string(got) != want {
		t.Errorf("stripFrontmatter(with frontmatter) = %q, want %q", got, want)
	}
	if strings.Contains(string(got), "---") {
		t.Errorf("stripped content should not contain a frontmatter delimiter: %q", got)
	}

	noFrontmatter := []byte("# Spekk Dev Loop\nno frontmatter here\n")
	if got := stripFrontmatter(noFrontmatter); string(got) != string(noFrontmatter) {
		t.Errorf("stripFrontmatter(no leading ---) = %q, want input unchanged %q", got, noFrontmatter)
	}
}

// TestInstall_Layout verifies the claude-code layout: the observer is an agent
// shim, and the coach, the builder, and the dev-loop are skills. It also checks
// that a re-install is idempotent.
func TestInstall_Layout(t *testing.T) {
	home := t.TempDir()
	opts := Options{Target: "claude-code", HomeDir: home, SkillFS: fakeSkillFS()}

	res, err := Install(opts)
	if err != nil {
		t.Fatal(err)
	}
	// 1 observer shim + 3 skills (coach, builder, dev-loop).
	if len(res.Written) != 4 {
		t.Fatalf("got %d files, want 4 (observer shim + coach, builder, dev-loop skills): %v", len(res.Written), res.Written)
	}

	// The observer stays an agent shim.
	obs := string(mustRead(t, filepath.Join(home, ".claude", "agents", "spekk-observer.md")))
	if !strings.HasPrefix(obs, "---\nname: spekk-observer\n") {
		t.Errorf("observer shim frontmatter should start with the name field: %q", obs)
	}
	if !strings.Contains(obs, "`spekk prompt observer`") {
		t.Errorf("observer shim body must run spekk prompt observer")
	}

	// The coach and builder are no longer agent shims.
	for _, role := range []string{"coach", "builder"} {
		if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "spekk-"+role+".md")); !os.IsNotExist(err) {
			t.Errorf("%s should not be an agent shim any more", role)
		}
	}

	// The coach and builder are thin skills.
	for _, role := range []string{"coach", "builder"} {
		c := string(mustRead(t, filepath.Join(home, ".claude", "skills", "spekk-"+role, "SKILL.md")))
		if !strings.Contains(c, "---\nname: spekk-"+role+"\n") {
			t.Errorf("%s skill should have the name frontmatter: %q", role, c)
		}
		if !strings.Contains(c, "`spekk prompt "+role+"`") {
			t.Errorf("%s skill body must run spekk prompt %s", role, role)
		}
	}

	// The dev-loop skill is present.
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")); err != nil {
		t.Errorf("dev-loop skill not written: %v", err)
	}

	// Idempotent: a re-install changes nothing.
	res2, err := Install(opts)
	if err != nil {
		t.Fatalf("re-install should succeed: %v", err)
	}
	if len(res2.Written) != 0 || len(res2.Removed) != 0 {
		t.Errorf("re-install not idempotent: written=%v removed=%v", res2.Written, res2.Removed)
	}
}

// TestInstall_Targets verifies per-target paths, extensions, and
// frontmatter variations for both global and project scopes.
func TestInstall_Targets(t *testing.T) {
	tests := []struct {
		target   string
		project  bool
		wantDir  []string // joined under home or cwd
		wantFile string
		contains string
		excludes string
		skillFS  fs.FS // non-nil for claude/claude-code targets (required by new logic)
	}{
		{"claude", false, []string{".claude", "agents"}, "spekk-observer.md", "name: spekk-observer", "", fakeSkillFS()},
		{"claude-code", true, []string{".claude", "agents"}, "spekk-observer.md", "", "", fakeSkillFS()},
		{"copilot", false, []string{".copilot", "agents"}, "spekk-observer.agent.md", "name: spekk-observer", "", nil},
		{"copilot", true, []string{".github", "agents"}, "spekk-observer.agent.md", "", "", fakeSkillFS()},
		{"cursor", false, []string{".cursor", "agents"}, "spekk-observer.md", "name: spekk-observer", "", fakeSkillFS()},
		{"cursor", true, []string{".cursor", "agents"}, "spekk-observer.md", "", "", fakeSkillFS()},
		{"opencode", false, []string{".config", "opencode", "agents"}, "spekk-observer.md", "mode: subagent", "name:", fakeSkillFS()},
		{"opencode", true, []string{".opencode", "agents"}, "spekk-observer.md", "", "", fakeSkillFS()},
		{"codex", false, []string{".codex", "prompts"}, "spekk-observer.md", "", "---", fakeSkillFS()},
	}

	for _, tt := range tests {
		name := tt.target
		if tt.project {
			name += "/project"
		}
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			opts := Options{Target: tt.target, Project: tt.project, SkillFS: tt.skillFS}
			if tt.project {
				opts.Cwd = base
			} else {
				opts.HomeDir = base
			}

			res, err := Install(opts)
			written := res.Written
			if err != nil {
				t.Fatal(err)
			}

			wantPath := filepath.Join(append([]string{base}, append(tt.wantDir, tt.wantFile)...)...)
			found := false
			for _, p := range written {
				if p == wantPath {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %s in written paths %v", wantPath, written)
			}

			data, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatal(err)
			}
			content := string(data)
			if tt.contains != "" && !strings.Contains(content, tt.contains) {
				t.Errorf("content should contain %q", tt.contains)
			}
			if tt.excludes != "" && strings.Contains(content, tt.excludes) {
				t.Errorf("content should not contain %q", tt.excludes)
			}
		})
	}
}

func TestInstall_Errors(t *testing.T) {
	// codex does not support project installs
	if _, err := Install(Options{Target: "codex", Project: true, Cwd: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "--project") {
		t.Errorf("codex --project should error explaining --project, got: %v", err)
	}

	// unknown target lists valid targets and the prompt fallback
	_, err := Install(Options{Target: "vim", HomeDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for unknown target")
	}
	for _, want := range ValidTargets() {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should list valid target %q, got: %v", want, err)
		}
	}
	if !strings.Contains(err.Error(), "spekk prompt") {
		t.Errorf("error should point at spekk prompt fallback, got: %v", err)
	}
}

// TestInstall_SkillFile covers native (unstripped) skill-writing: verbatim
// byte-equality in both scopes for claude-code, generalization to a second
// native target (opencode), the copilot opt-out, and the missing-FS error.
func TestInstall_SkillFile(t *testing.T) {
	skillContent := []byte("# spekk-dev-loop\nfake skill content for tests")
	skillFS := fstest.MapFS{
		skillEmbedPath: &fstest.MapFile{Data: skillContent},
	}

	t.Run("global writes skill to home/.claude/skills/", func(t *testing.T) {
		home := t.TempDir()
		res, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS})
		written := res.Written
		if err != nil {
			t.Fatal(err)
		}
		skillPath := filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
		found := false
		for _, p := range written {
			if p == skillPath {
				found = true
			}
		}
		if !found {
			t.Fatalf("skill path %s not in written %v", skillPath, written)
		}
		data, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("skill file not created: %v", err)
		}
		body, _, managed := ParseStamp(data)
		if !managed {
			t.Errorf("skill file is not stamped: %q", data)
		}
		if string(body) != string(skillContent) {
			t.Errorf("skill body mismatch: got %q, want %q", body, skillContent)
		}
	})

	t.Run("project writes skill to cwd/.claude/skills/", func(t *testing.T) {
		cwd := t.TempDir()
		res, err := Install(Options{Target: "claude-code", Project: true, Cwd: cwd, SkillFS: skillFS})
		written := res.Written
		if err != nil {
			t.Fatal(err)
		}
		skillPath := filepath.Join(cwd, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
		found := false
		for _, p := range written {
			if p == skillPath {
				found = true
			}
		}
		if !found {
			t.Fatalf("skill path %s not in written %v", skillPath, written)
		}
		data, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("skill file not created: %v", err)
		}
		body, _, managed := ParseStamp(data)
		if !managed {
			t.Errorf("skill file is not stamped: %q", data)
		}
		if string(body) != string(skillContent) {
			t.Errorf("skill body mismatch: got %q, want %q", body, skillContent)
		}
	})

	t.Run("copilot global writes coach and builder as agent shims (no skill path)", func(t *testing.T) {
		home := t.TempDir()
		res, err := Install(Options{Target: "copilot", HomeDir: home})
		if err != nil {
			t.Fatal(err)
		}
		// copilot global has no skill path, so the coach and builder fall back to
		// agent shims: 3 agent shims, no skill files.
		agentsDir := filepath.Join(home, ".copilot", "agents")
		for _, role := range []string{"observer", "coach", "builder"} {
			if _, err := os.Stat(filepath.Join(agentsDir, "spekk-"+role+".agent.md")); err != nil {
				t.Errorf("%s agent shim not written: %v", role, err)
			}
		}
		if len(res.Written) != 3 {
			t.Errorf("copilot global: got %d files, want 3 agent shims: %v", len(res.Written), res.Written)
		}
	})

	t.Run("opencode global writes skill to home/.config/opencode/skills/", func(t *testing.T) {
		home := t.TempDir()
		res, err := Install(Options{Target: "opencode", HomeDir: home, SkillFS: skillFS})
		written := res.Written
		if err != nil {
			t.Fatal(err)
		}
		skillPath := filepath.Join(home, ".config", "opencode", "skills", "spekk-dev-loop", "SKILL.md")
		found := false
		for _, p := range written {
			if p == skillPath {
				found = true
			}
		}
		if !found {
			t.Fatalf("skill path %s not in written %v", skillPath, written)
		}
		data, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("skill file not created: %v", err)
		}
		body, _, managed := ParseStamp(data)
		if !managed {
			t.Errorf("skill file is not stamped: %q", data)
		}
		if string(body) != string(skillContent) {
			t.Errorf("skill body mismatch: got %q, want %q", body, skillContent)
		}
	})

	t.Run("nil skill FS returns error for claude-code", func(t *testing.T) {
		// Ensure DefaultSkillFS is nil during this test.
		orig := DefaultSkillFS
		DefaultSkillFS = nil
		defer func() { DefaultSkillFS = orig }()

		home := t.TempDir()
		_, err := Install(Options{Target: "claude-code", HomeDir: home})
		if err == nil {
			t.Fatal("expected error when both SkillFS and DefaultSkillFS are nil")
		}
	})
}

// TestInstall_DevLoopCommand covers the frontmatter-stripped /spekk-dev-loop
// command written for cursor, codex, and copilot: the command/prompt
// harnesses that render a whole file as a prompt (and, for cursor, forbid
// YAML frontmatter outright).
func TestInstall_DevLoopCommand(t *testing.T) {
	skillFS := fakeSkillFSWithFrontmatter()

	assertStrippedFile := func(t *testing.T, path string) {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected dev-loop file at %s: %v", path, err)
		}
		content := string(data)
		if strings.Contains(content, "---") {
			t.Errorf("%s: content should have frontmatter stripped, got %q", path, content)
		}
		if !strings.HasPrefix(content, "# Spekk Dev Loop") {
			t.Errorf("%s: content should start with the stripped body, got %q", path, content)
		}
	}

	t.Run("cursor --project writes stripped command to cwd/.cursor/commands/", func(t *testing.T) {
		cwd := t.TempDir()
		if _, err := Install(Options{Target: "cursor", Project: true, Cwd: cwd, SkillFS: skillFS}); err != nil {
			t.Fatal(err)
		}
		assertStrippedFile(t, filepath.Join(cwd, ".cursor", "commands", "spekk-dev-loop.md"))
	})

	t.Run("codex global writes stripped prompt to home/.codex/prompts/", func(t *testing.T) {
		home := t.TempDir()
		if _, err := Install(Options{Target: "codex", HomeDir: home, SkillFS: skillFS}); err != nil {
			t.Fatal(err)
		}
		assertStrippedFile(t, filepath.Join(home, ".codex", "prompts", "spekk-dev-loop.md"))
	})
}

// TestInstall_MigratesUnstampedLegacyShim: an old unstamped coach agent shim
// (from a version before the reconciler) is backed up and removed, and the new
// coach skill is written.
func TestInstall_MigratesUnstampedLegacyShim(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(agentsDir, "spekk-coach.md")
	shim := []byte("---\nname: spekk-coach\n---\nYou are the spekk coach agent.\nRun `spekk prompt coach`.\n")
	if err := os.WriteFile(legacy, shim, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: fakeSkillFS()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy coach shim should be removed")
	}
	if _, err := os.Stat(legacy + ".bak"); err != nil {
		t.Errorf("legacy coach shim backup not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "spekk-coach", "SKILL.md")); err != nil {
		t.Errorf("new coach skill not written: %v", err)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "legacy agent shim") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a legacy-migration warning, got %v", res.Warnings)
	}
}

// TestInstall_PrunesStampedLegacyShim: a stamped coach or builder agent shim (a
// reconciler wrote it) is pruned on install, because the desired set no longer
// contains it.
func TestInstall_PrunesStampedLegacyShim(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(agentsDir, "spekk-builder.md")
	if err := os.WriteFile(legacy, StampContent([]byte("old builder shim")), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: fakeSkillFS()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("stamped legacy builder shim should be pruned")
	}
	found := false
	for _, p := range res.Removed {
		if p == legacy {
			found = true
		}
	}
	if !found {
		t.Errorf("removed list should include the pruned shim: %v", res.Removed)
	}
}

// TestInstall_LeavesUserFileAtLegacyPath: a file at a legacy path that is not a
// spekk shim is left alone.
func TestInstall_LeavesUserFileAtLegacyPath(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(agentsDir, "spekk-coach.md")
	if err := os.WriteFile(userFile, []byte("my own coach agent, not the tool\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: fakeSkillFS()}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(userFile); err != nil {
		t.Errorf("user file at legacy path should be left alone: %v", err)
	}
	if _, err := os.Stat(userFile + ".bak"); !os.IsNotExist(err) {
		t.Errorf("user file should not be backed up")
	}
}

// TestInstall_CommandHostStripsRoleSkill: a command host (codex) writes the
// coach skill with the frontmatter stripped.
func TestInstall_CommandHostStripsRoleSkill(t *testing.T) {
	home := t.TempDir()
	if _, err := Install(Options{Target: "codex", HomeDir: home, SkillFS: fakeSkillFS()}); err != nil {
		t.Fatal(err)
	}
	coach := string(mustRead(t, filepath.Join(home, ".codex", "prompts", "spekk-coach.md")))
	if strings.Contains(coach, "---") {
		t.Errorf("codex coach skill should have the frontmatter stripped: %q", coach)
	}
	if !strings.Contains(coach, "spekk prompt coach") {
		t.Errorf("codex coach skill body must run spekk prompt coach")
	}
}

// TestInstall_MigratesCodexSharedPathShim: for codex the legacy coach shim path
// equals the new coach skill path (the same prompts directory). The reconcile
// updates the file in place: it backs up the old shim and writes the stamped
// coach skill, and reports it in Written, not Removed.
func TestInstall_MigratesCodexSharedPathShim(t *testing.T) {
	home := t.TempDir()
	promptsDir := filepath.Join(home, ".codex", "prompts")
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(promptsDir, "spekk-coach.md")
	if err := os.WriteFile(shared, []byte("You are the spekk coach agent.\nRun `spekk prompt coach`.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Install(Options{Target: "codex", HomeDir: home, SkillFS: fakeSkillFS()})
	if err != nil {
		t.Fatal(err)
	}
	body, _, managed := ParseStamp(mustRead(t, shared))
	if !managed {
		t.Errorf("codex coach file should be stamped after migration")
	}
	if !strings.Contains(string(body), "spekk prompt coach") {
		t.Errorf("codex coach file should be the coach skill: %q", body)
	}
	inWritten := false
	for _, p := range res.Written {
		if p == shared {
			inWritten = true
		}
	}
	if !inWritten {
		t.Errorf("codex shared-path coach should be in Written: %v", res.Written)
	}
	for _, p := range res.Removed {
		if p == shared {
			t.Errorf("codex shared-path coach should not be in Removed")
		}
	}
	if _, err := os.Stat(shared + ".bak"); err != nil {
		t.Errorf("backup not written: %v", err)
	}
}
