package agent

import (
	"strings"
	"testing"
)

// The default harness must resolve to claude-code whether the name is empty
// (no --harness, no SPEKK_HARNESS) or given via the "claude" alias.
func TestResolveProfile_DefaultAndAlias(t *testing.T) {
	for _, name := range []string{"", "claude", "claude-code"} {
		p, err := ResolveProfile(name)
		if err != nil {
			t.Fatalf("ResolveProfile(%q) errored: %v", name, err)
		}
		if p.Name != "claude-code" || p.Binary != "claude" {
			t.Fatalf("ResolveProfile(%q) = %s/%s, want claude-code/claude", name, p.Name, p.Binary)
		}
	}
}

// An unknown harness must fail fast rather than return a zero profile a caller
// might spawn.
func TestResolveProfile_UnknownFailsFast(t *testing.T) {
	if _, err := ResolveProfile("nope"); err == nil {
		t.Fatal("ResolveProfile(\"nope\") should error")
	}
}

// The argv the default profile produces at each launch site must be
// byte-for-byte what the launch sites hardcoded before harnesses were
// selectable: interactive, the interactive-builder system prompt, and headless.
func TestDefaultProfile_ResolvedArgv(t *testing.T) {
	p := DefaultProfile()
	const msg = "activation message"

	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{"interactive", p.InteractiveArgs(msg), []string{"--dangerously-skip-permissions", msg}},
		{"system-prompt", p.SystemPromptArgs(msg), []string{"--dangerously-skip-permissions", "--system-prompt", msg}},
		{"headless", p.HeadlessArgs(msg), []string{"-p", "--dangerously-skip-permissions", msg}},
	}
	for _, tc := range cases {
		if !equalArgs(tc.got, tc.want) {
			t.Errorf("%s argv = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

// The not-found guidance must come from the profile and name that harness, so a
// non-claude harness never tells the user to install Claude Code.
func TestProfile_NotFoundNamesHarness(t *testing.T) {
	p := DefaultProfile()
	l1, l2 := p.notFoundLines()
	if !strings.Contains(l1, "Claude Code") {
		t.Errorf("first not-found line does not name the harness: %q", l1)
	}
	if !strings.Contains(l2, p.InstallURL) {
		t.Errorf("second not-found line does not carry the install URL: %q", l2)
	}

	other := Profile{DisplayName: "Widget", InstallURL: "https://widget.example"}
	o1, o2 := other.notFoundLines()
	if strings.Contains(o1+o2, "Claude") {
		t.Errorf("non-claude harness guidance mentions Claude: %q / %q", o1, o2)
	}
}

func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
