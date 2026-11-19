package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/cli"
	"github.com/spekk-ai/spekk-cli/internal/conversation"
)

const conversationDeliveryEnvVar = "SPEKK_CONVERSATION_DELIVERY"

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
  spekk conversation open --title <text> --body <text> [--severity <level>] [--channel <name|id>] [--attach <file> ...]

OPTIONS:
  --title <text>       Short summary of the request (required)
  --body <text>        Full details of the request (required)
  --severity <level>   One of: info, warning, critical (default: info)
  --channel <name|id>  Slack channel for the agent's bot (default: home channel)
  --attach <file>      Send a file in the same thread (repeat up to three times)
  --help, -h           Show this help message

This command works inside a sandbox session. If the app supports direct
delivery, the command returns the Slack message link and file IDs. An app
without direct delivery queues a plain message and cannot return a result or
attach files. Each file must be at most 10 MB, and all files together must
be at most 20 MB. Slack workspace file type rules apply.
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
	var paths []string
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] != "--attach" {
			filtered = append(filtered, args[i])
			continue
		}
		if i+1 == len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
			fmt.Fprintln(stderr, "Error: --attach requires a file")
			return 1
		}
		i++
		paths = append(paths, args[i])
	}
	if len(paths) > 3 {
		fmt.Fprintln(stderr, "Error: at most three files are allowed")
		return 1
	}
	flags := cli.ParseFlags(filtered, cli.FlagSet{
		"title":    {Names: []string{"--title"}, Type: cli.StringFlag},
		"body":     {Names: []string{"--body"}, Type: cli.StringFlag},
		"severity": {Names: []string{"--severity"}, Type: cli.StringFlag},
		"channel":  {Names: []string{"--channel"}, Type: cli.StringFlag},
		"help":     {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})

	if flags.Bool("help") {
		fmt.Fprint(stdout, conversationOpenUsage)
		return 0
	}
	if flags.Err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", flags.Err)
		return 1
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
	channel := strings.TrimSpace(flags.String("channel"))
	if flags.Count("channel") > 0 && channel == "" {
		fmt.Fprintln(stderr, "Error: --channel requires a non-empty channel name or ID")
		return 1
	}
	if severity == "" {
		severity = string(conversation.DefaultSeverity)
	} else if !conversation.IsValidSeverity(severity) {
		fmt.Fprintf(stderr, "Error: invalid --severity %q; valid values: %s, %s, %s\n",
			severity, conversation.SeverityInfo, conversation.SeverityWarning, conversation.SeverityCritical)
		return 1
	}

	sessionID := getenv("SPEKK_AGENT_SESSION_ID")
	if sessionID != "" && getenv(conversationDeliveryEnvVar) == "1" {
		resumeFile := getenv(conversation.SessionFileEnvVar)
		resumeID, err := os.ReadFile(resumeFile)
		if resumeFile == "" || err != nil || strings.TrimSpace(string(resumeID)) == "" {
			fmt.Fprintln(stderr, "Error: the active Claude session ID is not ready; try again")
			return 1
		}
		files := make([]uploadFile, 0, len(paths))
		var total int64
		for _, path := range paths {
			file, err := regularUpload(path, 10<<20)
			if err != nil {
				fmt.Fprintf(stderr, "Error: %s\n", err)
				return 1
			}
			total += file.size
			files = append(files, file)
		}
		if total > 20<<20 {
			fmt.Fprintln(stderr, "Error: all files together must be at most 20 MB")
			return 1
		}
		fields := map[string]string{"session_id": sessionID, "claude_session_id": strings.TrimSpace(string(resumeID)), "title": title, "body": body, "severity": severity}
		if channel != "" {
			fields["channel"] = channel
		}
		status, result, err := postAgentFiles(getenv, "/api/agents/conversations/", fields, files)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %s\n", err)
			return 1
		}
		link, _ := result["message_url"].(string)
		if status != 200 {
			if link != "" {
				fmt.Fprintf(stderr, "Error: text posted at %s; %s\n", link, responseText(result))
			} else {
				fmt.Fprintf(stderr, "Error: %s\n", responseText(result))
			}
			return 1
		}
		if link == "" {
			fmt.Fprintln(stderr, "Error: the app returned no message link")
			return 1
		}
		fmt.Fprintf(stdout, "Message: %s\n", link)
		if ids, ok := result["file_ids"].([]any); ok {
			for _, id := range ids {
				fmt.Fprintf(stdout, "File: %s\n", id)
			}
		}
		return 0
	}
	if len(paths) != 0 {
		if sessionID != "" {
			fmt.Fprintln(stderr, "Error: the app does not support conversation file delivery; update the app before using --attach")
			return 1
		}
		fmt.Fprintln(stderr, "Error: SPEKK_AGENT_SESSION_ID is not set; attachments need a current sandbox session")
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
		Channel:  channel,
	}
	if err := conversation.WriteRequest(spoolDir, req); err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "conversation request queued")
	return 0
}
