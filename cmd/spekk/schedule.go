package main

import (
	"fmt"
	"os"

	"github.com/spekk-ai/spekk-cli/internal/sandbox"
)

// scheduleHelpText is the help for `spekk schedule`.
const scheduleHelpText = `
spekk schedule - Manage the scheduled tasks of a sandbox agent

USAGE:
  spekk schedule list   <agent>
  spekk schedule create <agent> --slug <slug> --cron_expression <expr> --task_text <text> [fields]
  spekk schedule update <agent> <schedule> [fields]
  spekk schedule delete <agent> <schedule>
  spekk schedule run    <agent> <schedule>

SUBCOMMANDS:
  list    List the agent's schedules
  create  Create a schedule, and print it as JSON
  update  Change the given fields of a schedule, and print it as JSON
  delete  Delete a schedule
  run     Run a schedule now

ARGUMENTS:
  <agent>     A sandbox name from "spekk sandbox list", or an agent id
  <schedule>  A schedule id, or its slug

FIELDS (the flag names are the control host API field names):
  --slug <slug>                Stable job name, unique for the agent
  --cron_expression <expr>     Five-field cron expression, e.g. "0 9 * * *"
  --timezone <zone>            IANA zone the expression runs in (default: UTC)
  --task_text <text>           The message the agent receives
  --custom_context <text>      Context added to the session system prompt
  --role <role>                Session role (default: builder)
  --enabled true|false         A disabled schedule never fires

Every command needs SPEKK_API_TOKEN and SPEKK_HOST.
`

// runSchedule runs `spekk schedule <subcommand> ...`.
func runSchedule(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Print(scheduleHelpText)
		return
	}
	for _, a := range args[1:] {
		if a == "--help" || a == "-h" {
			fmt.Print(scheduleHelpText)
			return
		}
	}
	if err := sandbox.RunSchedule(args, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
