package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/cli"
	"github.com/spekk-ai/spekk-cli/internal/conversation"
)

// conversationUsage is the help text for `spekk conversation --help`.
const conversationUsage = `
spekk conversation - Request a conversation on the connected chat surface

USAGE:
  spekk conversation <subcommand> [options]

SUBCOMMANDS:
  open   Queue a request to open a conversation

OPTIONS:
  --help, -h  Show this help message

Use "spekk conversation open --help" for more information.
`

// conversationOpenUsage is the help text for `spekk conversation open --help`.
const conversationOpenUsage = `
spekk conversation open - Request that a conversation be opened on the
connected chat surface

USAGE:
  spekk conversation open --title <text> --body <text> [--severity <level>]

OPTIONS:
  --title <text>       Short summary of the request (required)
  --body <text>        Full details of the request (required)
  --severity <level>   One of: info, warning, critical (default: info)
  --help, -h           Show this help message

This command only works while running inside a sandbox session: it reads
the session's spool directory from the ` + conversation.SpoolEnvVar + `
environment variable, which the session sets up automatically. Running it
outside a sandbox session (spool var unset) fails with a clear error.

The command writes one request file into the spool directory and exits; it
does not open a connection itself and does not wait for a response.
`

// runConversation handles `spekk conversation <subcommand>`.
func runConversation(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Print(conversationUsage)
		if len(args) == 0 {
			os.Exit(1)
		}
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "open":
		runConversationOpen(subArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown conversation command: %s\n", subcommand)
		fmt.Fprintln(os.Stderr, `Run "spekk conversation --help" for available subcommands.`)
		os.Exit(1)
	}
}

// runConversationOpen implements `spekk conversation open`.
func runConversationOpen(args []string) {
	code := execConversationOpen(args, os.Stdout, os.Stderr, os.Getenv)
	if code != 0 {
		os.Exit(code)
	}
}

// execConversationOpen is the testable core of runConversationOpen. It
// writes messages to stdout/stderr and returns an exit code (0 = success).
// getenv is injected so tests can simulate the spool env var without
// mutating real process environment.
func execConversationOpen(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"title":    {Names: []string{"--title"}, Type: cli.StringFlag},
		"body":     {Names: []string{"--body"}, Type: cli.StringFlag},
		"severity": {Names: []string{"--severity"}, Type: cli.StringFlag},
		"help":     {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})

	if flags.Bool("help") {
		fmt.Fprint(stdout, conversationOpenUsage)
		return 0
	}

	title := flags.String("title")
	body := flags.String("body")

	var missing []string
	if title == "" {
		missing = append(missing, "--title")
	}
	if body == "" {
		missing = append(missing, "--body")
	}
	if len(missing) > 0 {
		fmt.Fprintf(stderr, "Error: missing required flag(s): %s\n", strings.Join(missing, ", "))
		return 1
	}

	severity := flags.String("severity")
	if severity == "" {
		severity = string(conversation.DefaultSeverity)
	} else if !conversation.IsValidSeverity(severity) {
		fmt.Fprintf(stderr, "Error: invalid --severity %q; valid values: %s, %s, %s\n",
			severity, conversation.SeverityInfo, conversation.SeverityWarning, conversation.SeverityCritical)
		return 1
	}

	spoolDir := getenv(conversation.SpoolEnvVar)
	if spoolDir == "" {
		fmt.Fprintf(stderr, "Error: %s is not set; \"spekk conversation open\" only works inside a sandbox session\n", conversation.SpoolEnvVar)
		return 1
	}

	req := conversation.Request{
		Title:    title,
		Body:     body,
		Severity: conversation.Severity(severity),
	}
	data, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(stderr, "Error: encoding request: %s\n", err)
		return 1
	}

	if err := writeRequestFile(spoolDir, data); err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "conversation request queued")
	return 0
}

// writeRequestFile atomically writes data as a new request file in spoolDir.
// It writes to a temp file first (created with a random component via
// os.CreateTemp, guaranteeing no collision with concurrent invocations) and
// renames it into place, so a concurrent drain of spoolDir never observes a
// partially written file.
func writeRequestFile(spoolDir string, data []byte) error {
	tmp, err := os.CreateTemp(spoolDir, "request-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating request file in %s: %w", spoolDir, err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing request file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing request file: %w", err)
	}

	// The final name reuses the same random component os.CreateTemp already
	// generated for tmpPath, so it is just as collision-proof.
	finalPath := tmpPath[:len(tmpPath)-len(".tmp")]
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("finalizing request file: %w", err)
	}

	return nil
}
