package install

import (
	"strings"
	"testing"
)

func TestParseArgs_DefaultLocalScope(t *testing.T) {
	opts, err := ParseArgs([]string{"coach", "meeting-notes"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Agent != "coach" {
		t.Errorf("agent: want coach, got %q", opts.Agent)
	}
	if opts.Skill != "meeting-notes" {
		t.Errorf("skill: want meeting-notes, got %q", opts.Skill)
	}
	if opts.Scope != ScopeLocal {
		t.Errorf("scope: want local, got %s", opts.Scope)
	}
	if opts.Source != "" {
		t.Errorf("source: want empty, got %q", opts.Source)
	}
	if opts.Force {
		t.Error("force: want false")
	}
}

func TestParseArgs_GlobalScope(t *testing.T) {
	opts, err := ParseArgs([]string{"builder", "my-skill", "--global"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Scope != ScopeGlobal {
		t.Errorf("scope: want global, got %s", opts.Scope)
	}
}

func TestParseArgs_LocalScopeExplicit(t *testing.T) {
	opts, err := ParseArgs([]string{"builder", "my-skill", "--local"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Scope != ScopeLocal {
		t.Errorf("scope: want local, got %s", opts.Scope)
	}
}

func TestParseArgs_SourceFlag(t *testing.T) {
	opts, err := ParseArgs([]string{"coach", "foo", "--source", "https://x.com/foo.md"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Source != "https://x.com/foo.md" {
		t.Errorf("source: want https://x.com/foo.md, got %q", opts.Source)
	}
	if opts.Agent != "coach" || opts.Skill != "foo" {
		t.Errorf("positionals lost: agent=%q skill=%q", opts.Agent, opts.Skill)
	}
}

func TestParseArgs_ForceFlag(t *testing.T) {
	opts, err := ParseArgs([]string{"coach", "foo", "--force"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Force {
		t.Error("force: want true")
	}
}

func TestParseArgs_GlobalAndLocalConflict(t *testing.T) {
	_, err := ParseArgs([]string{"coach", "foo", "--global", "--local"})
	if err == nil {
		t.Fatal("expected error for --global + --local, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--global") || !strings.Contains(msg, "--local") {
		t.Errorf("error should mention both flags, got: %s", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "mutually exclusive") {
		t.Errorf("error should explain mutual exclusion, got: %s", msg)
	}
}

func TestParseArgs_MissingAgent(t *testing.T) {
	_, err := ParseArgs([]string{})
	if err == nil {
		t.Fatal("expected error when <agent> is omitted")
	}
	if !strings.Contains(err.Error(), "agent") {
		t.Errorf("error should mention agent, got: %s", err.Error())
	}
}

func TestParseArgs_UnknownAgent(t *testing.T) {
	_, err := ParseArgs([]string{"bogus", "foo"})
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	msg := err.Error()
	for _, valid := range ValidAgents {
		if !strings.Contains(msg, valid) {
			t.Errorf("error should list valid agent %q, got: %s", valid, msg)
		}
	}
}

func TestParseArgs_Help(t *testing.T) {
	opts, err := ParseArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Help {
		t.Error("help: want true")
	}

	// UsageText should describe every flag listed in the assertion.
	wantFlags := []string{"--global", "--local", "--source", "--force", "--list", "--help"}
	for _, f := range wantFlags {
		if !strings.Contains(UsageText, f) {
			t.Errorf("UsageText missing %s", f)
		}
	}
}

func TestParseArgs_ListFlag(t *testing.T) {
	opts, err := ParseArgs([]string{"--list", "coach"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.List != "coach" {
		t.Errorf("list: want coach, got %q", opts.List)
	}
}

func TestParseArgs_ListFlagRejectsUnknownAgent(t *testing.T) {
	_, err := ParseArgs([]string{"--list", "bogus"})
	if err == nil {
		t.Fatal("expected error for --list bogus")
	}
}

func TestParseArgs_ListFlagRejectsStrayPositionals(t *testing.T) {
	// `spekk install --list coach foo` is almost certainly a typo; the `foo`
	// must not be silently dropped.
	_, err := ParseArgs([]string{"--list", "coach", "foo"})
	if err == nil {
		t.Fatal("expected error when --list is combined with positional args")
	}
	if !strings.Contains(err.Error(), "foo") {
		t.Errorf("error should name the stray positional, got: %s", err)
	}
}
