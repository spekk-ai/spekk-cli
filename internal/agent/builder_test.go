package agent

import (
	"testing"
)

func TestParseBuilderFlags_Once(t *testing.T) {
	cfg := ParseBuilderFlags([]string{"--once"})
	if !cfg.Once {
		t.Error("expected Once to be true")
	}
	if cfg.DryRun || cfg.Confirm || cfg.Interactive {
		t.Error("other flags should be false")
	}
}

func TestParseBuilderFlags_DryRunShort(t *testing.T) {
	cfg := ParseBuilderFlags([]string{"-d"})
	if !cfg.DryRun {
		t.Error("expected DryRun to be true")
	}
}

func TestParseBuilderFlags_Interactive(t *testing.T) {
	cfg := ParseBuilderFlags([]string{"-i"})
	if !cfg.Interactive {
		t.Error("expected Interactive to be true")
	}
}

func TestParseBuilderFlags_Confirm(t *testing.T) {
	cfg := ParseBuilderFlags([]string{"-c"})
	if !cfg.Confirm {
		t.Error("expected Confirm to be true")
	}
}

func TestParseBuilderFlags_SpecAndAssertion(t *testing.T) {
	cfg := ParseBuilderFlags([]string{"--spec", "auth", "--assertion", "login-flow"})
	if cfg.Spec != "auth" {
		t.Errorf("expected Spec=auth, got %s", cfg.Spec)
	}
	if cfg.Assertion != "login-flow" {
		t.Errorf("expected Assertion=login-flow, got %s", cfg.Assertion)
	}
}

func TestParseBuilderFlags_AllModes(t *testing.T) {
	cfg := ParseBuilderFlags([]string{"--once", "--confirm", "-s", "myspec"})
	if !cfg.Once {
		t.Error("expected Once")
	}
	if !cfg.Confirm {
		t.Error("expected Confirm")
	}
	if cfg.Spec != "myspec" {
		t.Errorf("expected Spec=myspec, got %s", cfg.Spec)
	}
}

func TestExtractSkillArg_SkillFirst(t *testing.T) {
	skill := ExtractSkillArg([]string{"my-skill", "--once"})
	if skill != "my-skill" {
		t.Errorf("expected my-skill, got %s", skill)
	}
}

func TestExtractSkillArg_SkillAfterFlags(t *testing.T) {
	skill := ExtractSkillArg([]string{"--spec", "auth", "my-skill"})
	if skill != "my-skill" {
		t.Errorf("expected my-skill, got %s", skill)
	}
}

func TestExtractSkillArg_NoSkill(t *testing.T) {
	skill := ExtractSkillArg([]string{"--once", "--dry-run"})
	if skill != "" {
		t.Errorf("expected empty string, got %s", skill)
	}
}

func TestExtractSkillArg_SkipsBoolFlag(t *testing.T) {
	skill := ExtractSkillArg([]string{"--once", "deploy"})
	if skill != "deploy" {
		t.Errorf("expected deploy, got %s", skill)
	}
}

func TestExtractSkillArg_SkipsStringFlagValue(t *testing.T) {
	// --spec takes a value, so "auth" should be skipped
	skill := ExtractSkillArg([]string{"--spec", "auth"})
	if skill != "" {
		t.Errorf("expected empty, got %s", skill)
	}
}

func TestExtractSkillArg_SkipsUnknownFlags(t *testing.T) {
	skill := ExtractSkillArg([]string{"--unknown", "my-skill"})
	if skill != "my-skill" {
		t.Errorf("expected my-skill, got %s", skill)
	}
}

func TestBuildSpekkNextCommand_NoFilters(t *testing.T) {
	args := BuildSpekkNextCommand("/usr/bin/spekk", BuilderConfig{})
	expected := []string{"/usr/bin/spekk", "next"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(args))
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("arg[%d]: expected %s, got %s", i, expected[i], a)
		}
	}
}

func TestBuildSpekkNextCommand_WithSpec(t *testing.T) {
	args := BuildSpekkNextCommand("spekk", BuilderConfig{Spec: "auth"})
	if len(args) != 4 || args[2] != "--spec" || args[3] != "auth" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestBuildSpekkNextCommand_WithAssertion(t *testing.T) {
	args := BuildSpekkNextCommand("spekk", BuilderConfig{Assertion: "login"})
	if len(args) != 4 || args[2] != "--assertion" || args[3] != "login" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestBuildSpekkNextCommand_WithBoth(t *testing.T) {
	args := BuildSpekkNextCommand("spekk", BuilderConfig{Spec: "auth", Assertion: "login"})
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d: %v", len(args), args)
	}
	if args[2] != "--spec" || args[3] != "auth" || args[4] != "--assertion" || args[5] != "login" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestHasHelp(t *testing.T) {
	tests := []struct {
		args   []string
		expect bool
	}{
		{[]string{"--help"}, true},
		{[]string{"-h"}, true},
		{[]string{"help"}, true},
		{[]string{"--once"}, false},
		{[]string{"--once", "-h"}, true},
		{[]string{}, false},
	}

	for _, tt := range tests {
		got := hasHelp(tt.args)
		if got != tt.expect {
			t.Errorf("hasHelp(%v) = %v, want %v", tt.args, got, tt.expect)
		}
	}
}
