package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstall_ShimContent verifies the full shim contract on the claude-code
// target, and that re-installing overwrites cleanly.
func TestInstall_ShimContent(t *testing.T) {
	home := t.TempDir()
	opts := Options{Target: "claude-code", HomeDir: home}

	written, err := Install(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 3 {
		t.Fatalf("got %d files, want 3 (coach, builder, observer)", len(written))
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
	}{
		{"claude", false, []string{".claude", "agents"}, "spekk-coach.md", "name: spekk-coach", ""},
		{"claude-code", true, []string{".claude", "agents"}, "spekk-coach.md", "", ""},
		{"copilot", false, []string{".copilot", "agents"}, "spekk-coach.agent.md", "name: spekk-coach", ""},
		{"copilot", true, []string{".github", "agents"}, "spekk-coach.agent.md", "", ""},
		{"cursor", false, []string{".cursor", "agents"}, "spekk-coach.md", "name: spekk-coach", ""},
		{"cursor", true, []string{".cursor", "agents"}, "spekk-coach.md", "", ""},
		{"opencode", false, []string{".config", "opencode", "agents"}, "spekk-coach.md", "mode: subagent", "name:"},
		{"opencode", true, []string{".opencode", "agents"}, "spekk-coach.md", "", ""},
		{"codex", false, []string{".codex", "prompts"}, "spekk-coach.md", "", "---"},
	}

	for _, tt := range tests {
		name := tt.target
		if tt.project {
			name += "/project"
		}
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			opts := Options{Target: tt.target, Project: tt.project}
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
