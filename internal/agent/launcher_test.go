package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// testEmbeddedFS creates a fake embedded FS with a base prompt for the agent.
func testEmbeddedFS(agent string) fstest.MapFS {
	return fstest.MapFS{
		"specs/" + agent + "-agent/" + agent + ".prompt.md": {
			Data: []byte("# " + agent + " base prompt"),
		},
	}
}

func TestBuildActivationMessage_Coach(t *testing.T) {
	cli.DefaultEmbeddedFS = testEmbeddedFS("coach")
	defer func() { cli.DefaultEmbeddedFS = nil }()

	msg, err := BuildActivationMessage(LaunchOptions{
		Agent: "coach",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(msg, "You are the Coach Agent") {
		t.Error("should contain capitalized agent name")
	}
	if !strings.Contains(msg, "# coach base prompt") {
		t.Error("should contain prompt content")
	}
	if !strings.Contains(msg, "Working directory:") {
		t.Error("should contain working directory")
	}
}

func TestBuildActivationMessage_WithExtraMessage(t *testing.T) {
	cli.DefaultEmbeddedFS = testEmbeddedFS("coach")
	defer func() { cli.DefaultEmbeddedFS = nil }()

	msg, err := BuildActivationMessage(LaunchOptions{
		Agent:        "coach",
		ExtraMessage: "\n\n## Extra Content",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(msg, "## Extra Content") {
		t.Error("should append extra message")
	}
}

func TestBuildSkillMessage_WithSkill(t *testing.T) {
	install := t.TempDir()

	// Create package skill
	skillDir := filepath.Join(install, "specs", "coach-skills-system")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "my-skill.md"), []byte("# Skill Content"), 0o644)

	msg, err := BuildSkillMessage(install, "coach", "my-skill", []string{"my-skill"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(msg, "Skill Activation") {
		t.Error("should contain skill activation header")
	}
	if !strings.Contains(msg, "<skill-content>") {
		t.Error("should contain skill content tags")
	}
	if !strings.Contains(msg, "# Skill Content") {
		t.Error("should contain skill file content")
	}
}

func TestBuildSkillMessage_NotFound(t *testing.T) {
	install := t.TempDir()

	msg, err := BuildSkillMessage(install, "coach", "nonexistent", []string{"nonexistent"})
	if err != nil {
		t.Fatal(err)
	}
	if msg != "" {
		t.Errorf("expected empty string for missing skill, got: %s", msg)
	}
}

func TestBuildSkillMessage_MeetingWithTranscript(t *testing.T) {
	install := t.TempDir()

	// Create meeting skill
	skillDir := filepath.Join(install, "specs", "coach-skills-system")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "meeting-notes-to-specs-skill.md"),
		[]byte("# Meeting Skill"), 0o644)

	// Create transcript file
	transcriptFile := filepath.Join(t.TempDir(), "notes.txt")
	os.WriteFile(transcriptFile, []byte("Meeting notes content here"), 0o644)

	msg, err := BuildSkillMessage(install, "coach", "meeting",
		[]string{"meeting", transcriptFile})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(msg, "<transcript>") {
		t.Error("should contain transcript tags")
	}
	if !strings.Contains(msg, "Meeting notes content here") {
		t.Error("should contain transcript content")
	}
	if !strings.Contains(msg, "Process this transcript now") {
		t.Error("should instruct to process transcript")
	}
}

func TestBuildSkillMessage_MeetingWithoutTranscript(t *testing.T) {
	install := t.TempDir()

	skillDir := filepath.Join(install, "specs", "coach-skills-system")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "meeting-notes-to-specs-skill.md"),
		[]byte("# Meeting Skill"), 0o644)

	msg, err := BuildSkillMessage(install, "coach", "meeting", []string{"meeting"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(msg, "No transcript file was provided") {
		t.Error("should indicate no transcript provided")
	}
}

func TestBuildSkillMessage_MeetingMissingFile(t *testing.T) {
	install := t.TempDir()

	skillDir := filepath.Join(install, "specs", "coach-skills-system")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "meeting-notes-to-specs-skill.md"),
		[]byte("# Meeting Skill"), 0o644)

	_, err := BuildSkillMessage(install, "coach", "meeting",
		[]string{"meeting", "/nonexistent/file.txt"})
	if err == nil {
		t.Fatal("expected error for missing transcript file")
	}
	if !strings.Contains(err.Error(), "Transcript file not found") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestBuildSkillMessage_MeetingPathTraversal(t *testing.T) {
	install := t.TempDir()

	skillDir := filepath.Join(install, "specs", "coach-skills-system")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "meeting-notes-to-specs-skill.md"),
		[]byte("# Meeting Skill"), 0o644)

	_, err := BuildSkillMessage(install, "coach", "meeting",
		[]string{"meeting", "../../etc/passwd"})
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "resolves outside working directory") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestBuildActivationMessage_UnknownAgent(t *testing.T) {
	_, err := BuildActivationMessage(LaunchOptions{
		Agent: "unknown",
	})
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

// TestBuildHelpText_ObserverWithSkills verifies observer help includes
// the dynamic skill list AND observer-specific options/examples.
func TestBuildHelpText_ObserverWithSkills(t *testing.T) {
	install := t.TempDir()
	skillDir := filepath.Join(install, "specs", "observer-skills")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "coverage-gap.md"), []byte("# Coverage Gap"), 0o644)

	// Isolate cwd/home so local/global layers don't shadow.
	isolated := t.TempDir()
	t.Setenv("HOME", isolated)
	origWd, _ := os.Getwd()
	os.Chdir(isolated)
	t.Cleanup(func() { os.Chdir(origWd) })

	out := buildHelpText(install, "observer")

	if !strings.Contains(out, "AVAILABLE SKILLS:") {
		t.Error("help should contain AVAILABLE SKILLS section")
	}
	if !strings.Contains(out, "coverage-gap") {
		t.Error("help should list discovered observer skill")
	}
	if !strings.Contains(out, "--interval") {
		t.Error("help should document observer's --interval option")
	}
	if !strings.Contains(out, "--quiet") {
		t.Error("help should document observer's --quiet option")
	}
	if !strings.Contains(out, "spekk observer") {
		t.Error("help should reference the observer command")
	}
}

// TestBuildHelpText_ObserverNoSkills verifies the no-skills fallback.
func TestBuildHelpText_ObserverNoSkills(t *testing.T) {
	install := t.TempDir()

	isolated := t.TempDir()
	t.Setenv("HOME", isolated)
	origWd, _ := os.Getwd()
	os.Chdir(isolated)
	t.Cleanup(func() { os.Chdir(origWd) })

	out := buildHelpText(install, "observer")

	if !strings.Contains(out, "(none found)") {
		t.Error("help should show (none found) when no skills exist")
	}
	if !strings.Contains(out, "--interval") || !strings.Contains(out, "--quiet") {
		t.Error("observer-specific options must appear even when no skills exist")
	}
}

// TestBuildHelpText_ObserverSkillsDeduped verifies skills shadowed by a
// higher-priority layer appear only once.
func TestBuildHelpText_ObserverSkillsDeduped(t *testing.T) {
	install := t.TempDir()
	pkgDir := filepath.Join(install, "specs", "observer-skills")
	os.MkdirAll(pkgDir, 0o755)
	os.WriteFile(filepath.Join(pkgDir, "coverage-gap.md"), []byte("# pkg"), 0o644)

	isolated := t.TempDir()
	t.Setenv("HOME", isolated)
	origWd, _ := os.Getwd()
	os.Chdir(isolated)
	t.Cleanup(func() { os.Chdir(origWd) })

	// Local layer shadows the package layer for the same skill name.
	localDir := filepath.Join(isolated, ".spekk", "skills", "observer")
	os.MkdirAll(localDir, 0o755)
	os.WriteFile(filepath.Join(localDir, "coverage-gap.md"), []byte("# local"), 0o644)

	out := buildHelpText(install, "observer")
	// Isolate the AVAILABLE SKILLS section — the EXAMPLES section may also
	// reference the skill name, which is not a duplicate listing.
	skillsSection := out
	if idx := strings.Index(out, "AVAILABLE SKILLS:"); idx >= 0 {
		skillsSection = out[idx:]
	}
	if end := strings.Index(skillsSection, "OPTIONS:"); end >= 0 {
		skillsSection = skillsSection[:end]
	}
	if strings.Count(skillsSection, "coverage-gap") != 1 {
		t.Errorf("expected coverage-gap to appear once in skills section, got %d:\n%s",
			strings.Count(skillsSection, "coverage-gap"), skillsSection)
	}
}
