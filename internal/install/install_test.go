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

// TestInstall_ShimContent verifies the full shim contract on the claude-code
// target, and that re-installing overwrites cleanly.
func TestInstall_ShimContent(t *testing.T) {
	home := t.TempDir()
	opts := Options{Target: "claude-code", HomeDir: home, SkillFS: fakeSkillFS()}

	written, err := Install(opts)
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
		{"copilot", true, []string{".github", "agents"}, "spekk-coach.agent.md", "", "", nil},
		{"cursor", false, []string{".cursor", "agents"}, "spekk-coach.md", "name: spekk-coach", "", nil},
		{"cursor", true, []string{".cursor", "agents"}, "spekk-coach.md", "", "", nil},
		{"opencode", false, []string{".config", "opencode", "agents"}, "spekk-coach.md", "mode: subagent", "name:", nil},
		{"opencode", true, []string{".opencode", "agents"}, "spekk-coach.md", "", "", nil},
		{"codex", false, []string{".codex", "prompts"}, "spekk-coach.md", "", "---", nil},
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

			written, err := Install(opts)
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

// TestInstall_SkillFile covers all skill-writing behavior for the claude-code target.
func TestInstall_SkillFile(t *testing.T) {
	skillContent := []byte("# spekk-dev-loop\nfake skill content for tests")
	skillFS := fstest.MapFS{
		skillEmbedPath: &fstest.MapFile{Data: skillContent},
	}

	t.Run("global writes skill to home/.claude/skills/", func(t *testing.T) {
		home := t.TempDir()
		written, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS})
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
		if string(data) != string(skillContent) {
			t.Errorf("skill bytes mismatch: got %q, want %q", data, skillContent)
		}
	})

	t.Run("project writes skill to cwd/.claude/skills/", func(t *testing.T) {
		cwd := t.TempDir()
		written, err := Install(Options{Target: "claude-code", Project: true, Cwd: cwd, SkillFS: skillFS})
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
		if string(data) != string(skillContent) {
			t.Errorf("skill bytes mismatch: got %q, want %q", data, skillContent)
		}
	})

	t.Run("claude alias behaves identically to claude-code", func(t *testing.T) {
		home := t.TempDir()
		written, err := Install(Options{Target: "claude", HomeDir: home, SkillFS: skillFS})
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
			t.Fatalf("claude alias: skill path %s not in written %v", skillPath, written)
		}
	})

	t.Run("non-claude-code target produces no skill file", func(t *testing.T) {
		home := t.TempDir()
		written, err := Install(Options{Target: "cursor", HomeDir: home})
		if err != nil {
			t.Fatal(err)
		}
		// Only the 3 shim files; no SKILL.md
		if len(written) != 3 {
			t.Fatalf("cursor: got %d written paths, want 3", len(written))
		}
		skillDir := filepath.Join(home, ".claude", "skills")
		if _, err := os.Stat(skillDir); err == nil {
			t.Errorf("cursor install should not create .claude/skills/ dir")
		}
	})

	t.Run("re-install overwrites existing SKILL.md without error", func(t *testing.T) {
		home := t.TempDir()
		opts := Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}
		if _, err := Install(opts); err != nil {
			t.Fatal(err)
		}
		// Second install must succeed and overwrite
		if _, err := Install(opts); err != nil {
			t.Fatalf("re-install should succeed: %v", err)
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
