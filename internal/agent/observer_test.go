package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

func TestParseObserverFlags_Defaults(t *testing.T) {
	cfg, err := ParseObserverFlags([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interval != 0 {
		t.Errorf("expected Interval=0, got %d", cfg.Interval)
	}
	if cfg.Quiet {
		t.Error("expected Quiet=false")
	}
}

func TestParseObserverFlags_Interval(t *testing.T) {
	cfg, err := ParseObserverFlags([]string{"--interval", "60"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interval != 60 {
		t.Errorf("expected Interval=60, got %d", cfg.Interval)
	}
}

func TestParseObserverFlags_Quiet(t *testing.T) {
	cfg, err := ParseObserverFlags([]string{"--quiet"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Quiet {
		t.Error("expected Quiet=true")
	}
}

func TestParseObserverFlags_InvalidInterval(t *testing.T) {
	_, err := ParseObserverFlags([]string{"--interval", "abc"})
	if err == nil {
		t.Fatal("expected error for non-numeric interval")
	}
	if !strings.Contains(err.Error(), "positive number") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestParseObserverFlags_NegativeInterval(t *testing.T) {
	_, err := ParseObserverFlags([]string{"--interval", "-5"})
	if err == nil {
		t.Fatal("expected error for negative interval")
	}
}

func TestParseObserverFlags_Both(t *testing.T) {
	cfg, err := ParseObserverFlags([]string{"--quiet", "--interval", "30"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interval != 30 {
		t.Errorf("expected Interval=30, got %d", cfg.Interval)
	}
	if !cfg.Quiet {
		t.Error("expected Quiet=true")
	}
}

func TestBuildObserverOptionsMessage_NoOptions(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{})
	if msg != "" {
		t.Errorf("expected empty string, got: %s", msg)
	}
}

func TestBuildObserverOptionsMessage_IntervalOnly(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{Interval: 60})
	if !strings.Contains(msg, "Scan interval: 60 seconds") {
		t.Error("should contain interval")
	}
	if strings.Contains(msg, "Quiet mode") {
		t.Error("should not contain quiet mode")
	}
}

func TestBuildObserverOptionsMessage_QuietOnly(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{Quiet: true})
	if !strings.Contains(msg, "Quiet mode: enabled") {
		t.Error("should contain quiet mode")
	}
	if strings.Contains(msg, "Scan interval") {
		t.Error("should not contain interval")
	}
}

func TestBuildObserverOptionsMessage_Both(t *testing.T) {
	msg := BuildObserverOptionsMessage(ObserverConfig{Interval: 30, Quiet: true})
	if !strings.Contains(msg, "Scan interval: 30 seconds") {
		t.Error("should contain interval")
	}
	if !strings.Contains(msg, "Quiet mode: enabled") {
		t.Error("should contain quiet mode")
	}
	if !strings.Contains(msg, "CLI Options provided:") {
		t.Error("should contain header")
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
	isolated := t.TempDir()
	t.Setenv("HOME", isolated)
	origWd, _ := os.Getwd()
	if err := os.Chdir(isolated); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })

	return install
}

func TestExtractSkillArgFromFlagSet_ObserverFlags_SkillFirst(t *testing.T) {
	skill := ExtractSkillArgFromFlagSet([]string{"coverage-gap", "--interval", "60"}, ObserverFlags)
	if skill != "coverage-gap" {
		t.Errorf("expected coverage-gap, got %q", skill)
	}
}

func TestExtractSkillArgFromFlagSet_ObserverFlags_SkipsIntervalValue(t *testing.T) {
	skill := ExtractSkillArgFromFlagSet([]string{"--interval", "60"}, ObserverFlags)
	if skill != "" {
		t.Errorf("expected empty (interval value skipped), got %q", skill)
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
		{"flag only", []string{"--interval", "60"}},
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

			if _, err := ParseObserverFlags(tc.args); err != nil {
				t.Errorf("flag parsing should succeed for fallback path: %v", err)
			}
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
			if !strings.Contains(out, "--interval") || !strings.Contains(out, "--quiet") {
				t.Errorf("%s: help missing observer-specific options", flag)
			}
			if !hasHelp([]string{flag}) {
				t.Errorf("hasHelp should detect %q", flag)
			}
		})
	}
}
