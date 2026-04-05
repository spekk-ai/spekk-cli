package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// setupDirs creates temporary directories for install, work, and home and
// returns a configured PromptResolver plus a cleanup function.
func setupDirs(t *testing.T) (*PromptResolver, func()) {
	t.Helper()
	tmp := t.TempDir()
	installDir := filepath.Join(tmp, "install")
	workDir := filepath.Join(tmp, "work")
	homeDir := filepath.Join(tmp, "home")

	for _, d := range []string{installDir, workDir, homeDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	pr := &PromptResolver{
		InstallDir: installDir,
		WorkDir:    workDir,
		HomeDir:    homeDir,
	}

	return pr, func() {} // t.TempDir handles cleanup
}

// packageBasePath returns the expected path for a package base prompt file.
func packageBasePath(pr *PromptResolver, agent string) string {
	return filepath.Join(pr.InstallDir, "specs", agent+"-agent", agent+".prompt.md")
}

// localOverridePath returns the path for a local override prompt file.
func localOverridePath(pr *PromptResolver, agent string) string {
	return filepath.Join(pr.WorkDir, ".spekk", agent+".prompt.override.md")
}

// globalOverridePath returns the path for a global override prompt file.
func globalOverridePath(pr *PromptResolver, agent string) string {
	return filepath.Join(pr.HomeDir, ".spekk", agent+".prompt.override.md")
}

// localExtendPath returns the path for a local extend prompt file.
func localExtendPath(pr *PromptResolver, agent string) string {
	return filepath.Join(pr.WorkDir, ".spekk", agent+".prompt.md")
}

// globalExtendPath returns the path for a global extend prompt file.
func globalExtendPath(pr *PromptResolver, agent string) string {
	return filepath.Join(pr.HomeDir, ".spekk", agent+".prompt.md")
}

// ---------------------------------------------------------------------------
// ResolvePrompt – unknown agent
// ---------------------------------------------------------------------------

func TestResolvePrompt_UnknownAgent(t *testing.T) {
	pr, _ := setupDirs(t)
	_, err := pr.ResolvePrompt("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("expected 'unknown agent' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – package base prompt
// ---------------------------------------------------------------------------

func TestResolvePrompt_PackageBase(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "builder"), "Base builder prompt")

	result, err := pr.ResolvePrompt("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Base builder prompt" {
		t.Errorf("expected base prompt, got %q", result)
	}
}

func TestResolvePrompt_PackageBaseMissing(t *testing.T) {
	pr, _ := setupDirs(t)
	_, err := pr.ResolvePrompt("coach")
	if err == nil {
		t.Fatal("expected error for missing package base prompt")
	}
	if !strings.Contains(err.Error(), "prompt file not found") {
		t.Errorf("expected 'prompt file not found' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – local override takes priority
// ---------------------------------------------------------------------------

func TestResolvePrompt_LocalOverrideTakesPriority(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "coach"), "Package base")
	writeFile(t, globalOverridePath(pr, "coach"), "Global override")
	writeFile(t, localOverridePath(pr, "coach"), "Local override")

	result, err := pr.ResolvePrompt("coach")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Local override" {
		t.Errorf("expected 'Local override', got %q", result)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – global override takes priority over package base
// ---------------------------------------------------------------------------

func TestResolvePrompt_GlobalOverrideTakesPriority(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "coach"), "Package base")
	writeFile(t, globalOverridePath(pr, "coach"), "Global override")

	result, err := pr.ResolvePrompt("coach")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Global override" {
		t.Errorf("expected 'Global override', got %q", result)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – local override without package base
// ---------------------------------------------------------------------------

func TestResolvePrompt_LocalOverrideWithoutPackageBase(t *testing.T) {
	pr, _ := setupDirs(t)
	// No package base prompt file exists
	writeFile(t, localOverridePath(pr, "builder"), "Local override only")

	result, err := pr.ResolvePrompt("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Local override only" {
		t.Errorf("expected 'Local override only', got %q", result)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – extension layers appended
// ---------------------------------------------------------------------------

func TestResolvePrompt_ExtensionLayersAppended(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "builder"), "Base")
	writeFile(t, globalExtendPath(pr, "builder"), "Global extend")
	writeFile(t, localExtendPath(pr, "builder"), "Local extend")

	result, err := pr.ResolvePrompt("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Base" + PromptSeparator + "Global extend" + PromptSeparator + "Local extend"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestResolvePrompt_OnlyGlobalExtend(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "observer"), "Base")
	writeFile(t, globalExtendPath(pr, "observer"), "Global extend")

	result, err := pr.ResolvePrompt("observer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Base" + PromptSeparator + "Global extend"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestResolvePrompt_OnlyLocalExtend(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "coach"), "Base")
	writeFile(t, localExtendPath(pr, "coach"), "Local extend")

	result, err := pr.ResolvePrompt("coach")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Base" + PromptSeparator + "Local extend"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – override with extensions
// ---------------------------------------------------------------------------

func TestResolvePrompt_OverrideWithExtensions(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "builder"), "Package base (unused)")
	writeFile(t, localOverridePath(pr, "builder"), "Local override")
	writeFile(t, globalExtendPath(pr, "builder"), "Global extend")
	writeFile(t, localExtendPath(pr, "builder"), "Local extend")

	result, err := pr.ResolvePrompt("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Local override" + PromptSeparator + "Global extend" + PromptSeparator + "Local extend"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – missing extensions silently skipped
// ---------------------------------------------------------------------------

func TestResolvePrompt_MissingExtensionsSilentlySkipped(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "coach"), "Base prompt")

	result, err := pr.ResolvePrompt("coach")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No extensions, so result should just be the base.
	if result != "Base prompt" {
		t.Errorf("expected 'Base prompt', got %q", result)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – works for all agents
// ---------------------------------------------------------------------------

func TestResolvePrompt_AllAgents(t *testing.T) {
	for _, agent := range []string{"coach", "builder", "observer"} {
		t.Run(agent, func(t *testing.T) {
			pr, _ := setupDirs(t)
			writeFile(t, packageBasePath(pr, agent), agent+" base prompt")

			result, err := pr.ResolvePrompt(agent)
			if err != nil {
				t.Fatalf("unexpected error for agent %s: %v", agent, err)
			}
			if result != agent+" base prompt" {
				t.Errorf("expected %q, got %q", agent+" base prompt", result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – separator format
// ---------------------------------------------------------------------------

func TestResolvePrompt_SeparatorFormat(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "builder"), "A")
	writeFile(t, globalExtendPath(pr, "builder"), "B")
	writeFile(t, localExtendPath(pr, "builder"), "C")

	result, err := pr.ResolvePrompt("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "A\n\n---\n\nB\n\n---\n\nC" {
		t.Errorf("unexpected separator format, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// CreateActivationMessage – basic
// ---------------------------------------------------------------------------

func TestCreateActivationMessage_Basic(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "builder"), "You are the builder.")

	msg, err := pr.CreateActivationMessage("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(msg, "You are the Builder Agent") {
		t.Errorf("expected agent display name, got: %s", msg)
	}
	if !strings.Contains(msg, "Working directory: "+pr.WorkDir) {
		t.Errorf("expected working directory in message")
	}
	if !strings.Contains(msg, "Spekk installation: "+pr.InstallDir) {
		t.Errorf("expected installation directory in message")
	}
	if !strings.Contains(msg, "You are the builder.") {
		t.Errorf("expected prompt content in message")
	}
}

// ---------------------------------------------------------------------------
// CreateActivationMessage – capitalises agent name
// ---------------------------------------------------------------------------

func TestCreateActivationMessage_Capitalisation(t *testing.T) {
	for _, tc := range []struct {
		agent   string
		display string
	}{
		{"coach", "Coach"},
		{"builder", "Builder"},
		{"observer", "Observer"},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			pr, _ := setupDirs(t)
			writeFile(t, packageBasePath(pr, tc.agent), "prompt")

			msg, err := pr.CreateActivationMessage(tc.agent)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expected := "You are the " + tc.display + " Agent"
			if !strings.Contains(msg, expected) {
				t.Errorf("expected %q in message, got: %s", expected, msg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CreateActivationMessage – error on missing prompt
// ---------------------------------------------------------------------------

func TestCreateActivationMessage_ErrorOnMissingPrompt(t *testing.T) {
	pr, _ := setupDirs(t)
	_, err := pr.CreateActivationMessage("builder")
	if err == nil {
		t.Fatal("expected error for missing prompt")
	}
	if !strings.Contains(err.Error(), "error loading prompt for builder") {
		t.Errorf("expected wrapped error message, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateActivationMessage – unknown agent
// ---------------------------------------------------------------------------

func TestCreateActivationMessage_UnknownAgent(t *testing.T) {
	pr, _ := setupDirs(t)
	_, err := pr.CreateActivationMessage("unknown")
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("expected 'unknown agent' in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateActivationMessage – includes layered prompt
// ---------------------------------------------------------------------------

func TestCreateActivationMessage_IncludesLayers(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "coach"), "Base coach")
	writeFile(t, localExtendPath(pr, "coach"), "Extra instructions")

	msg, err := pr.CreateActivationMessage("coach")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(msg, "Base coach") {
		t.Errorf("expected base prompt in activation message")
	}
	if !strings.Contains(msg, "Extra instructions") {
		t.Errorf("expected extension in activation message")
	}
	if !strings.Contains(msg, PromptSeparator) {
		t.Errorf("expected separator between layers in activation message")
	}
}

// ---------------------------------------------------------------------------
// readIfExists – internal helper
// ---------------------------------------------------------------------------

func TestReadIfExists_FileExists(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	content, err := readIfExists(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "hello" {
		t.Errorf("expected 'hello', got %q", content)
	}
}

func TestReadIfExists_FileMissing(t *testing.T) {
	content, err := readIfExists("/nonexistent/path/abc.txt")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty string for missing file, got %q", content)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – global override without package base
// ---------------------------------------------------------------------------

func TestResolvePrompt_GlobalOverrideWithoutPackageBase(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, globalOverridePath(pr, "observer"), "Global override only")

	result, err := pr.ResolvePrompt("observer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Global override only" {
		t.Errorf("expected 'Global override only', got %q", result)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – priority chain: local > global > package
// ---------------------------------------------------------------------------

func TestResolvePrompt_PriorityChain(t *testing.T) {
	pr, _ := setupDirs(t)

	// Start with only package base.
	writeFile(t, packageBasePath(pr, "builder"), "Package")

	result, err := pr.ResolvePrompt("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Package" {
		t.Errorf("step 1: expected 'Package', got %q", result)
	}

	// Add global override -> should win over package base.
	writeFile(t, globalOverridePath(pr, "builder"), "Global")

	result, err = pr.ResolvePrompt("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Global" {
		t.Errorf("step 2: expected 'Global', got %q", result)
	}

	// Add local override -> should win over global override.
	writeFile(t, localOverridePath(pr, "builder"), "Local")

	result, err = pr.ResolvePrompt("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Local" {
		t.Errorf("step 3: expected 'Local', got %q", result)
	}
}

// ---------------------------------------------------------------------------
// ResolvePrompt – extension order: global extend then local extend
// ---------------------------------------------------------------------------

func TestResolvePrompt_ExtensionOrder(t *testing.T) {
	pr, _ := setupDirs(t)
	writeFile(t, packageBasePath(pr, "coach"), "Base")
	writeFile(t, localExtendPath(pr, "coach"), "Local ext")
	writeFile(t, globalExtendPath(pr, "coach"), "Global ext")

	result, err := pr.ResolvePrompt("coach")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Global extend comes before local extend.
	parts := strings.Split(result, PromptSeparator)
	if len(parts) != 3 {
		t.Fatalf("expected 3 layers, got %d: %v", len(parts), parts)
	}
	if parts[0] != "Base" {
		t.Errorf("layer 0: expected 'Base', got %q", parts[0])
	}
	if parts[1] != "Global ext" {
		t.Errorf("layer 1: expected 'Global ext', got %q", parts[1])
	}
	if parts[2] != "Local ext" {
		t.Errorf("layer 2: expected 'Local ext', got %q", parts[2])
	}
}
