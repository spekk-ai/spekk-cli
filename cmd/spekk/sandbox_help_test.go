package main

import (
	"strings"
	"testing"
)

// TestSandboxHelpListsEverySubcommand pins the sandbox help to the
// subcommands launchSandbox dispatches, so a new one cannot ship unlisted.
func TestSandboxHelpListsEverySubcommand(t *testing.T) {
	for _, cmd := range []string{"create", "provision", "list", "status", "ssh", "destroy", "deploy"} {
		if !strings.Contains(sandboxHelpText, "\n  "+cmd+" ") {
			t.Errorf("sandboxHelpText missing subcommand %q:\n%s", cmd, sandboxHelpText)
		}
		if !strings.Contains(helpText, cmd) {
			t.Errorf("top-level helpText does not name sandbox subcommand %q", cmd)
		}
	}
	for _, want := range []string{"<name>", "--auth", "--force"} {
		if !strings.Contains(sandboxProvisionHelpText, want) {
			t.Errorf("sandboxProvisionHelpText missing %q", want)
		}
	}
}

// TestSandboxPositionalSkipsFlagValues checks that a subcommand finds its
// <name> whichever side of a value-taking flag it sits on.
func TestSandboxPositionalSkipsFlagValues(t *testing.T) {
	cases := map[string][]string{
		"box": {"--auth", "subscription", "box", "--force"},
		"":    {"--auth", "subscription", "--force"},
	}
	cases["also-box"] = []string{"also-box", "--auth", "bedrock"}
	for want, args := range cases {
		if got := sandboxPositional(args, "--auth"); got != want {
			t.Errorf("sandboxPositional(%q) = %q, want %q", args, got, want)
		}
	}
}
