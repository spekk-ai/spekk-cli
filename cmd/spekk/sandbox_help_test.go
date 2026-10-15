package main

import (
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/sandbox"
)

// TestSandboxHelpListsEverySubcommand pins the sandbox help to the
// subcommands launchSandbox dispatches, so a new one cannot ship unlisted.
func TestSandboxHelpListsEverySubcommand(t *testing.T) {
	for _, cmd := range []string{"create", "provision", "register", "list", "status", "ssh", "destroy", "deploy"} {
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

// TestSandboxHelpNamesKindFlags pins the create and list help to the flags
// that mark and find one-off sandboxes.
func TestSandboxHelpNamesKindFlags(t *testing.T) {
	for _, want := range []string{"--one-off", "--expires", "--purpose"} {
		if !strings.Contains(sandboxCreateHelpText, "\n  "+want+" ") {
			t.Errorf("sandboxCreateHelpText missing %q", want)
		}
	}
	for _, want := range []string{"--expired", "--all-hosts"} {
		if !strings.Contains(sandboxListHelpText, "\n  "+want+" ") {
			t.Errorf("sandboxListHelpText missing %q", want)
		}
	}
}

// TestScheduleHelpListsEverySubcommand pins the schedule help to the
// subcommands RunSchedule accepts, and to the API field names.
func TestScheduleHelpListsEverySubcommand(t *testing.T) {
	for _, cmd := range []string{"list", "create", "update", "delete", "run"} {
		if !strings.Contains(scheduleHelpText, "\n  "+cmd+" ") {
			t.Errorf("scheduleHelpText missing subcommand %q", cmd)
		}
	}
	for _, field := range sandbox.ScheduleFields {
		if !strings.Contains(scheduleHelpText, "--"+field+" ") {
			t.Errorf("scheduleHelpText missing --%s", field)
		}
	}
	if !strings.Contains(helpText, "\n  schedule ") {
		t.Error("top-level helpText does not list schedule")
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
