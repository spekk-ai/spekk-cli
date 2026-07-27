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

// TestInstall_ShimContent verifies the full shim contract on the claude-code
// target, and that re-installing overwrites cleanly.
func TestInstall_ShimContent(t *testing.T) {
	home := t.TempDir()
	opts := Options{Target: "claude-code", HomeDir: home, SkillFS: fakeSkillFS()}

	res, err := Install(opts)
	written := res.Written
	if err != nil {
		t.Fatal(err)
	}
	// 3 shims (coach, builder, observer) + 1 skill file
	if len(written) != 4 {
		t.Fatalf("got %d files, want 4 (coach, builder, observer shims + skill)", len(written))
	}

	for _, agent := range []string{"coach", "builder", "observer"} {
		path := filepath.Join(home, ".claude", "agents", "spekk-"+agent+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
		content := string(data)
		if !strings.HasPrefix(content, "---\nname: spekk-"+agent+"\n") {
			t.Errorf("%s: frontmatter should start with name field", agent)
		}
		if !strings.Contains(content, `description: "`) {
			t.Errorf("%s: description must be a quoted YAML scalar", agent)
		}
		if !strings.Contains(content, "specs/ directory") {
			t.Errorf("%s: description should scope delegation to specs/ projects", agent)
		}
		if !strings.Contains(content, "`spekk prompt "+agent+"`") {
			t.Errorf("%s: body must instruct running spekk prompt", agent)
		}
		if !strings.Contains(content, "https://github.com/spekk-ai/spekk-cli") {
			t.Errorf("%s: body must link install instructions for missing binary", agent)
		}
	}

	// Skill file must also be in the returned slice
	skillPath := filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
	found := false
	for _, p := range written {
		if p == skillPath {
			found = true
		}
	}
	if !found {
		t.Errorf("skill path %s not in written %v", skillPath, written)
	}

	// Idempotent: re-install overwrites without error
	if _, err := Install(opts); err != nil {
		t.Fatalf("re-install should succeed: %v", err)
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
		{"claude", false, []string{".claude", "agents"}, "spekk-coach.md", "name: spekk-coach", "", fakeSkillFS()},
		{"claude-code", true, []string{".claude", "agents"}, "spekk-coach.md", "", "", fakeSkillFS()},
		{"copilot", false, []string{".copilot", "agents"}, "spekk-coach.agent.md", "name: spekk-coach", "", nil},
		{"copilot", true, []string{".github", "agents"}, "spekk-coach.agent.md", "", "", fakeSkillFS()},
		{"cursor", false, []string{".cursor", "agents"}, "spekk-coach.md", "name: spekk-coach", "", fakeSkillFS()},
		{"cursor", true, []string{".cursor", "agents"}, "spekk-coach.md", "", "", fakeSkillFS()},
		{"opencode", false, []string{".config", "opencode", "agents"}, "spekk-coach.md", "mode: subagent", "name:", fakeSkillFS()},
		{"opencode", true, []string{".opencode", "agents"}, "spekk-coach.md", "", "", fakeSkillFS()},
		{"codex", false, []string{".codex", "prompts"}, "spekk-coach.md", "", "---", fakeSkillFS()},
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

	t.Run("copilot global produces no dev-loop file at all (genuinely opted out)", func(t *testing.T) {
		home := t.TempDir()
		res, err := Install(Options{Target: "copilot", HomeDir: home})
		written := res.Written
		if err != nil {
			t.Fatal(err)
		}
		// Only the 3 shim files; no dev-loop file, no SkillFS needed.
		if len(written) != 3 {
			t.Fatalf("copilot global: got %d written paths, want 3, got %v", len(written), written)
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
