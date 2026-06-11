package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallClaudeCodeGlobal(t *testing.T) {
	home := t.TempDir()
	written, err := Install(Options{Target: "claude-code", HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 3 {
		t.Fatalf("got %d files, want 3", len(written))
	}

	wantDir := filepath.Join(home, ".claude", "agents")
	for _, agent := range []string{"coach", "builder", "observer"} {
		path := filepath.Join(wantDir, "spekk-"+agent+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
		content := string(data)
		if !strings.HasPrefix(content, "---\nname: spekk-"+agent+"\n") {
			t.Errorf("%s: frontmatter should start with name field, got:\n%s", agent, content[:80])
		}
		if !strings.Contains(content, "description: ") {
			t.Errorf("%s: missing description", agent)
		}
		if !strings.Contains(content, "specs/ directory") {
			t.Errorf("%s: description should scope to specs/ directory", agent)
		}
		if !strings.Contains(content, "`spekk prompt "+agent+"`") {
			t.Errorf("%s: body must instruct running spekk prompt", agent)
		}
		if !strings.Contains(content, "https://github.com/spekk-ai/spekk-cli") {
			t.Errorf("%s: body must link install instructions for missing binary", agent)
		}
	}
}

func TestInstallClaudeAlias(t *testing.T) {
	home := t.TempDir()
	written, err := Install(Options{Target: "claude", HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(written[0], filepath.Join(".claude", "agents")) {
		t.Errorf("alias claude should resolve to claude-code paths, got %s", written[0])
	}
}

func TestInstallClaudeCodeProject(t *testing.T) {
	cwd := t.TempDir()
	written, err := Install(Options{Target: "claude-code", Project: true, Cwd: cwd})
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(cwd, ".claude", "agents")
	for _, path := range written {
		if filepath.Dir(path) != wantDir {
			t.Errorf("project install wrote to %s, want dir %s", path, wantDir)
		}
	}
}

func TestInstallOpenCode(t *testing.T) {
	home := t.TempDir()
	written, err := Install(Options{Target: "opencode", HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(home, ".config", "opencode", "agents")
	if filepath.Dir(written[0]) != wantDir {
		t.Fatalf("wrote to %s, want dir %s", written[0], wantDir)
	}
	data, _ := os.ReadFile(written[0])
	content := string(data)
	if !strings.Contains(content, "mode: subagent") {
		t.Error("opencode frontmatter must set mode: subagent")
	}
	if strings.Contains(content, "name:") {
		t.Error("opencode frontmatter should not set name (filename is the agent name)")
	}
}

func TestInstallCodexGlobal(t *testing.T) {
	home := t.TempDir()
	written, err := Install(Options{Target: "codex", HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(home, ".codex", "prompts")
	if filepath.Dir(written[0]) != wantDir {
		t.Fatalf("wrote to %s, want dir %s", written[0], wantDir)
	}
	data, _ := os.ReadFile(written[0])
	if strings.HasPrefix(string(data), "---") {
		t.Error("codex shims should have no frontmatter")
	}
}

func TestInstallCodexProjectUnsupported(t *testing.T) {
	_, err := Install(Options{Target: "codex", Project: true, Cwd: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for codex --project")
	}
	if !strings.Contains(err.Error(), "--project") {
		t.Errorf("error should explain --project is unsupported, got: %v", err)
	}
}

func TestInstallUnknownTarget(t *testing.T) {
	_, err := Install(Options{Target: "vim", HomeDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for unknown target")
	}
	for _, want := range ValidTargets() {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should list valid target %q, got: %v", want, err)
		}
	}
}

func TestInstallIdempotent(t *testing.T) {
	home := t.TempDir()
	if _, err := Install(Options{Target: "claude-code", HomeDir: home}); err != nil {
		t.Fatal(err)
	}
	// Second install overwrites without error
	written, err := Install(Options{Target: "claude-code", HomeDir: home})
	if err != nil {
		t.Fatalf("re-install should succeed: %v", err)
	}
	if len(written) != 3 {
		t.Fatalf("got %d files, want 3", len(written))
	}
}
