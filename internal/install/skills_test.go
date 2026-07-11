package install

import (
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// `spekk skills list <agent>` should render the resolver's ListSkills result
// with one row per skill, each carrying the source directory (or "(embedded)")
// so users can tell which scope is shadowing what.
func TestFormatSkillsList_RendersNameAndSource(t *testing.T) {
	skills := []cli.SkillEntry{
		{Name: "local-skill", Source: "/proj/.spekk/skills/coach"},
		{Name: "global-skill", Source: "/home/u/.spekk/skills/coach"},
		{Name: "embedded-skill", Source: "(embedded)"},
	}
	out := FormatSkillsList("coach", skills)

	if !strings.Contains(out, "Skills for coach") {
		t.Errorf("missing header for agent, got:\n%s", out)
	}
	for _, s := range skills {
		if !strings.Contains(out, s.Name) {
			t.Errorf("output missing skill name %q:\n%s", s.Name, out)
		}
		if !strings.Contains(out, s.Source) {
			t.Errorf("output missing source %q for skill %q:\n%s", s.Source, s.Name, out)
		}
	}
}

// When no skills resolve for the agent, the list command must still exit 0
// after printing a clear "no skills found" message — scripts should not be
// forced to distinguish empty from error.
func TestFormatSkillsList_EmptyMessage(t *testing.T) {
	out := FormatSkillsList("observer", nil)
	if !strings.Contains(strings.ToLower(out), "no skills found") {
		t.Errorf("empty case should say 'no skills found', got: %s", out)
	}
	if !strings.Contains(out, "observer") {
		t.Errorf("empty message should name the agent, got: %s", out)
	}
}

// Agent validation must mirror `spekk install` — unknown agents are rejected
// with an error that lists the valid options, so the two surfaces stay in
// lockstep on what `<agent>` means.
func TestValidateSkillsAgent(t *testing.T) {
	for _, a := range ValidAgents {
		if err := ValidateSkillsAgent(a); err != nil {
			t.Errorf("valid agent %q rejected: %v", a, err)
		}
	}
	err := ValidateSkillsAgent("nope")
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	msg := err.Error()
	if !strings.Contains(msg, "nope") {
		t.Errorf("error should name the bad agent, got: %s", msg)
	}
	for _, a := range ValidAgents {
		if !strings.Contains(msg, a) {
			t.Errorf("error should list valid agent %q, got: %s", a, msg)
		}
	}
}

// `spekk skills` (no subcommand) must print usage that documents the `list`
// subcommand — this is the discoverability hook for new users.
func TestSkillsUsageText_DocumentsList(t *testing.T) {
	if !strings.Contains(SkillsUsageText, "list") {
		t.Errorf("usage text must mention 'list' subcommand:\n%s", SkillsUsageText)
	}
	if !strings.Contains(SkillsUsageText, "<agent>") {
		t.Errorf("usage text must mention <agent> argument:\n%s", SkillsUsageText)
	}
}
