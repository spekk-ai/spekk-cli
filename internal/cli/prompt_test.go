package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestDirs creates temp home and cwd directories, plus a fake install dir
// with a base prompt file for the given agent.
func setupTestDirs(t *testing.T, agent string) (homeDir, cwd, installDir string) {
	t.Helper()
	homeDir = t.TempDir()
	cwd = t.TempDir()
	installDir = t.TempDir()

	// Create base prompt
	agentDir := filepath.Join(installDir, "specs", agent+"-agent")
	os.MkdirAll(agentDir, 0o755)
	os.WriteFile(
		filepath.Join(agentDir, agent+".prompt.md"),
		[]byte("# Base "+agent+" prompt"),
		0o644,
	)
	return
}

func newResolver(home, cwd, install string) *PromptResolver {
	return &PromptResolver{HomeDir: home, Cwd: cwd, InstallDir: install}
}

func TestGetPromptContent_BasePrompt(t *testing.T) {
	home, cwd, install := setupTestDirs(t, "builder")
	r := newResolver(home, cwd, install)

	content, err := r.GetPromptContent("builder")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "# Base builder prompt") {
		t.Errorf("expected base prompt, got: %s", content)
	}
	if strings.Contains(content, "---") {
		t.Error("should not contain separator when no layers appended")
	}
}

func TestGetPromptContent_UnknownAgent(t *testing.T) {
	r := newResolver(t.TempDir(), t.TempDir(), t.TempDir())

	_, err := r.GetPromptContent("unknown-agent")
	if err == nil || !strings.Contains(err.Error(), "Unknown agent") {
		t.Errorf("expected Unknown agent error, got: %v", err)
	}
}

func TestGetPromptContent_MissingBasePrompt(t *testing.T) {
	r := newResolver(t.TempDir(), t.TempDir(), t.TempDir())

	_, err := r.GetPromptContent("builder")
	if err == nil || !strings.Contains(err.Error(), "Prompt file not found") {
		t.Errorf("expected Prompt file not found error, got: %v", err)
	}
}

func TestGetPromptContent_GlobalOverride(t *testing.T) {
	home, cwd, install := setupTestDirs(t, "builder")

	globalDir := filepath.Join(home, ".spekk")
	os.MkdirAll(globalDir, 0o755)
	os.WriteFile(filepath.Join(globalDir, "builder.prompt.override.md"), []byte("# Global Override"), 0o644)

	r := newResolver(home, cwd, install)
	content, err := r.GetPromptContent("builder")
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Global Override" {
		t.Errorf("expected global override, got: %s", content)
	}
}

func TestGetPromptContent_LocalOverrideTakesPrecedence(t *testing.T) {
	home, cwd, install := setupTestDirs(t, "builder")

	globalDir := filepath.Join(home, ".spekk")
	os.MkdirAll(globalDir, 0o755)
	os.WriteFile(filepath.Join(globalDir, "builder.prompt.override.md"), []byte("# Global Override"), 0o644)

	localDir := filepath.Join(cwd, ".spekk")
	os.MkdirAll(localDir, 0o755)
	os.WriteFile(filepath.Join(localDir, "builder.prompt.override.md"), []byte("# Local Override"), 0o644)

	r := newResolver(home, cwd, install)
	content, err := r.GetPromptContent("builder")
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Local Override" {
		t.Errorf("expected local override, got: %s", content)
	}
}

func TestGetPromptContent_GlobalExtendAppended(t *testing.T) {
	home, cwd, install := setupTestDirs(t, "builder")

	globalDir := filepath.Join(home, ".spekk")
	os.MkdirAll(globalDir, 0o755)
	os.WriteFile(filepath.Join(globalDir, "builder.prompt.md"), []byte("## Global Extend"), 0o644)

	r := newResolver(home, cwd, install)
	content, err := r.GetPromptContent("builder")
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(content, "\n\n---\n\n")
	if len(parts) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(parts))
	}
	if !strings.Contains(parts[0], "# Base builder prompt") {
		t.Error("first layer should be base prompt")
	}
	if !strings.Contains(parts[1], "## Global Extend") {
		t.Error("second layer should be global extend")
	}
}

func TestGetPromptContent_LocalExtendAppended(t *testing.T) {
	home, cwd, install := setupTestDirs(t, "builder")

	localDir := filepath.Join(cwd, ".spekk")
	os.MkdirAll(localDir, 0o755)
	os.WriteFile(filepath.Join(localDir, "builder.prompt.md"), []byte("## Local Extend"), 0o644)

	r := newResolver(home, cwd, install)
	content, err := r.GetPromptContent("builder")
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(content, "\n\n---\n\n")
	if len(parts) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(parts))
	}
	if !strings.Contains(parts[1], "## Local Extend") {
		t.Error("second layer should be local extend")
	}
}

func TestGetPromptContent_AllThreeLayers(t *testing.T) {
	home, cwd, install := setupTestDirs(t, "builder")

	globalDir := filepath.Join(home, ".spekk")
	os.MkdirAll(globalDir, 0o755)
	os.WriteFile(filepath.Join(globalDir, "builder.prompt.md"), []byte("## Global Extend"), 0o644)

	localDir := filepath.Join(cwd, ".spekk")
	os.MkdirAll(localDir, 0o755)
	os.WriteFile(filepath.Join(localDir, "builder.prompt.md"), []byte("## Local Extend"), 0o644)

	r := newResolver(home, cwd, install)
	content, err := r.GetPromptContent("builder")
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(content, "\n\n---\n\n")
	if len(parts) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(parts))
	}
	if !strings.Contains(parts[0], "# Base builder prompt") {
		t.Error("first layer should be base")
	}
	if !strings.Contains(parts[1], "## Global Extend") {
		t.Error("second layer should be global extend")
	}
	if !strings.Contains(parts[2], "## Local Extend") {
		t.Error("third layer should be local extend")
	}
}

func TestGetPromptContent_GlobalOverrideWithExtends(t *testing.T) {
	home, cwd, install := setupTestDirs(t, "builder")

	globalDir := filepath.Join(home, ".spekk")
	os.MkdirAll(globalDir, 0o755)
	os.WriteFile(filepath.Join(globalDir, "builder.prompt.override.md"), []byte("# Custom Base"), 0o644)
	os.WriteFile(filepath.Join(globalDir, "builder.prompt.md"), []byte("## Extra Instructions"), 0o644)

	r := newResolver(home, cwd, install)
	content, err := r.GetPromptContent("builder")
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(content, "\n\n---\n\n")
	if len(parts) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(parts))
	}
	if parts[0] != "# Custom Base" {
		t.Errorf("first layer should be override, got: %s", parts[0])
	}
	if parts[1] != "## Extra Instructions" {
		t.Errorf("second layer should be extend, got: %s", parts[1])
	}
}

func TestGetPromptContent_MissingBaseWithGlobalOverride(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	install := t.TempDir() // No base prompt

	globalDir := filepath.Join(home, ".spekk")
	os.MkdirAll(globalDir, 0o755)
	os.WriteFile(filepath.Join(globalDir, "builder.prompt.override.md"), []byte("# Override Builder"), 0o644)

	r := newResolver(home, cwd, install)
	content, err := r.GetPromptContent("builder")
	if err != nil {
		t.Fatalf("should succeed with override: %v", err)
	}
	if !strings.Contains(content, "# Override Builder") {
		t.Errorf("expected override content, got: %s", content)
	}
}

func TestGetPromptContent_WorksForAllAgents(t *testing.T) {
	for _, agent := range []string{"coach", "builder", "observer"} {
		t.Run(agent, func(t *testing.T) {
			home, cwd, install := setupTestDirs(t, agent)

			globalDir := filepath.Join(home, ".spekk")
			os.MkdirAll(globalDir, 0o755)
			os.WriteFile(filepath.Join(globalDir, agent+".prompt.md"), []byte("## Global "+agent), 0o644)

			r := newResolver(home, cwd, install)
			content, err := r.GetPromptContent(agent)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(content, "## Global "+agent) {
				t.Errorf("should include global extend for %s", agent)
			}
			if !strings.Contains(content, "\n\n---\n\n") {
				t.Errorf("should contain separator for %s", agent)
			}
		})
	}
}

func TestCreateActivationMessage(t *testing.T) {
	home, cwd, install := setupTestDirs(t, "builder")
	r := newResolver(home, cwd, install)

	msg, err := r.CreateActivationMessage("builder")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(msg, "You are the Builder Agent") {
		t.Error("should capitalize agent name")
	}
	if !strings.Contains(msg, "Working directory:") {
		t.Error("should include working directory")
	}
	if !strings.Contains(msg, "Spekk installation:") {
		t.Error("should include installation path")
	}
	if !strings.Contains(msg, "# Base builder prompt") {
		t.Error("should include prompt content")
	}
}
