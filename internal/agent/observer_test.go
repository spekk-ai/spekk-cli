package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

func TestParseObserverFlags_Defaults(t *testing.T) {
	cfg := ParseObserverFlags([]string{})
	if cfg.Quiet {
		t.Error("expected Quiet=false")
	}
}

func TestParseObserverFlags_Quiet(t *testing.T) {
	cfg := ParseObserverFlags([]string{"--quiet"})
	if !cfg.Quiet {
		t.Error("expected Quiet=true")
	}
}

// The scan-interval flag is gone: it set a cadence inside a session that ran
// until it was stopped, and a run now ends on its own.
func TestParseObserverFlags_IntervalIsNoLongerAFlag(t *testing.T) {
	if _, defined := ObserverFlags["interval"]; defined {
		t.Error("--interval is still defined; cadence belongs to the schedule")
	}
}

// Removing it from the flag set is not enough on its own. The first bare
// positional argument is read as a skill name, so `--interval 60` would parse
// and then launch a skill called "60" -- the wrong thing, quietly, for anyone
// whose script still carries the flag. It has to be recognised and refused.
func TestRemovedObserverFlag_CatchesTheOldIntervalFlag(t *testing.T) {
	// The misfire this guards against is real, so assert it is reachable.
	if skill := ExtractSkillArgFromFlagSet([]string{"--interval", "60"}, ObserverFlags); skill != "60" {
		t.Fatalf("expected the stale value to look like a skill name, got %q", skill)
	}

	for _, args := range [][]string{
		{"--interval", "60"},
		{"--interval=60"},
		{"--quiet", "--interval", "60"},
	} {
		flag, message := RemovedObserverFlag(args)
		if flag != "--interval" {
			t.Errorf("args %v: expected --interval to be refused, got %q", args, flag)
		}
		if !strings.Contains(message, "install-cron") {
			t.Errorf("args %v: the message must say where cadence lives now: %q", args, message)
		}
	}

	if flag, _ := RemovedObserverFlag([]string{"--quiet", "coverage-gap"}); flag != "" {
		t.Errorf("a supported invocation was refused: %q", flag)
	}
}

func TestParseObserverFlags_QuietWithHeadless(t *testing.T) {
	cfg := ParseObserverFlags([]string{"--quiet", "--headless"})
	if !cfg.Quiet {
		t.Error("expected Quiet=true")
	}
	if !cfg.Headless {
		t.Error("expected Headless=true")
	}
}

func TestBuildObserverOptionsMessage_NoOptions(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{})
	if msg != "" {
		t.Errorf("expected empty string, got: %s", msg)
	}
}

func TestBuildObserverOptionsMessage_Quiet(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{Quiet: true})
	if !strings.Contains(msg, "Quiet mode: enabled") {
		t.Error("should contain quiet mode")
	}
	if !strings.Contains(msg, "CLI Options provided:") {
		t.Error("should contain header")
	}
}

// The activation message must not describe a cadence to the agent. A run
// files one observation and ends, so an interval read as a
// standing instruction would contradict the prompt it is appended to.
func TestBuildObserverOptionsMessage_NamesNoCadence(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{Quiet: true})
	for _, banned := range []string{"interval", "Scan interval", "seconds"} {
		if strings.Contains(msg, banned) {
			t.Errorf("activation message still names a cadence (%q):\n%s", banned, msg)
		}
	}
}

// setupObserverSkill creates a package observer skill in installDir and changes
// cwd to a temp directory so local/global skill dirs don't accidentally shadow.
func setupObserverSkill(t *testing.T, skillName, content string) string {
	t.Helper()
	install := t.TempDir()
	skillDir := filepath.Join(install, "specs", "observer-skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, skillName+".md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Isolate cwd/home so local/global layers don't pick up real skills.
	// XDG_CONFIG_HOME must be pinned too: config.GlobalConfigDir prefers it
	// over HOME, so a developer machine with it set would leak real skills.
	isolated := t.TempDir()
	t.Setenv("HOME", isolated)
	t.Setenv("XDG_CONFIG_HOME", isolated)
	origWd, _ := os.Getwd()
	if err := os.Chdir(isolated); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })

	return install
}

func TestExtractSkillArgFromFlagSet_ObserverFlags_SkillFirst(t *testing.T) {
	skill := ExtractSkillArgFromFlagSet([]string{"coverage-gap", "--quiet"}, ObserverFlags)
	if skill != "coverage-gap" {
		t.Errorf("expected coverage-gap, got %q", skill)
	}
}

// A value belonging to a flag is skipped, so it is never read as a skill.
func TestExtractSkillArgFromFlagSet_ObserverFlags_SkipsFlagValue(t *testing.T) {
	skill := ExtractSkillArgFromFlagSet([]string{"--claude-path", "/usr/bin/claude"}, ObserverFlags)
	if skill != "" {
		t.Errorf("expected empty (flag value skipped), got %q", skill)
	}
}

func TestExtractSkillArgFromFlagSet_ObserverFlags_SkipsQuietBool(t *testing.T) {
	skill := ExtractSkillArgFromFlagSet([]string{"--quiet", "coverage-gap"}, ObserverFlags)
	if skill != "coverage-gap" {
		t.Errorf("expected coverage-gap, got %q", skill)
	}
}

func TestExtractSkillArgFromFlagSet_ObserverFlags_NoPositional(t *testing.T) {
	skill := ExtractSkillArgFromFlagSet([]string{"--quiet"}, ObserverFlags)
	if skill != "" {
		t.Errorf("expected empty, got %q", skill)
	}
}

func TestObserverSkillResolution_Found(t *testing.T) {
	install := setupObserverSkill(t, "coverage-gap", "# Coverage Gap Skill\nBody")

	args := []string{"coverage-gap"}
	skillName := ExtractSkillArgFromFlagSet(args, ObserverFlags)
	if skillName != "coverage-gap" {
		t.Fatalf("expected coverage-gap, got %q", skillName)
	}

	sr := &cli.SkillResolver{
		GlobalConfigDir: os.Getenv("HOME"),
		Cwd:             mustGetwd(t),
		InstallDir:      install,
	}
	resolved := sr.ResolveSkill("observer", skillName)
	if resolved == nil {
		t.Fatal("expected ResolveSkill to find coverage-gap")
	}

	msg, err := BuildSkillMessage(install, "observer", skillName, args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "Skill Activation: `spekk observer coverage-gap`") {
		t.Error("expected skill activation header for observer")
	}
	if !strings.Contains(msg, "<skill-content>") || !strings.Contains(msg, "</skill-content>") {
		t.Error("expected skill content wrapped in <skill-content> tags")
	}
	if !strings.Contains(msg, "# Coverage Gap Skill") {
		t.Error("expected skill body to be inlined")
	}
}

func TestObserverSkillResolution_NotFound(t *testing.T) {
	install := setupObserverSkill(t, "coverage-gap", "# Coverage Gap Skill")

	cases := []struct {
		name string
		args []string
	}{
		{"flag only", []string{"--claude-path", "/usr/bin/claude"}},
		{"unknown skill name", []string{"does-not-exist"}},
		{"empty args", []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			skillName := ExtractSkillArgFromFlagSet(tc.args, ObserverFlags)

			sr := &cli.SkillResolver{
				GlobalConfigDir: os.Getenv("HOME"),
				Cwd:             mustGetwd(t),
				InstallDir:      install,
			}
			if skillName != "" && sr.ResolveSkill("observer", skillName) != nil {
				t.Fatalf("expected no skill match for args %v", tc.args)
			}

			ParseObserverFlags(tc.args)
		})
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

func TestObserverHelp_UsesSharedHelper(t *testing.T) {
	install := setupObserverSkill(t, "coverage-gap", "# Coverage Gap")

	for _, flag := range []string{"--help", "-h", "help"} {
		t.Run(flag, func(t *testing.T) {
			out := buildHelpText(install, "observer")
			if !strings.Contains(out, "AVAILABLE SKILLS:") {
				t.Errorf("%s: help missing AVAILABLE SKILLS section", flag)
			}
			if !strings.Contains(out, "coverage-gap") {
				t.Errorf("%s: help missing dynamically discovered skill", flag)
			}
			if !strings.Contains(out, "--quiet") {
				t.Errorf("%s: help missing observer-specific options", flag)
			}
			if !hasHelp([]string{flag}) {
				t.Errorf("hasHelp should detect %q", flag)
			}
		})
	}
}

func TestObserverLockFile_DefaultLoop(t *testing.T) {
	got := ObserverLockFile("/proj", "")
	want := filepath.Join("/proj", ".spekk", "observer-loop.lock")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestObserverLockFile_PerSkill(t *testing.T) {
	got := ObserverLockFile("/proj", "workflow-progress")
	want := filepath.Join("/proj", ".spekk", "observer-workflow-progress.lock")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestObserverLockFile_ConsolidateKeepsItsPath(t *testing.T) {
	got := ObserverLockFile("/proj", "consolidate")
	want := filepath.Join("/proj", ".spekk", "observer-consolidate.lock")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestObserverLockFile_SanitizesUnsafeNames(t *testing.T) {
	got := ObserverLockFile("/proj", "../evil/skill name")
	want := filepath.Join("/proj", ".spekk", "observer-..-evil-skill-name.lock")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
	if filepath.Dir(got) != filepath.Join("/proj", ".spekk") {
		t.Errorf("lock file escaped .spekk/: %q", got)
	}
}
