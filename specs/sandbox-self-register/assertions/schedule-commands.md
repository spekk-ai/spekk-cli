---
id: schedule-commands
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: done
branch: feat/sandbox-self-register
depends-on: control-host-client
---

# `spekk schedule` Manages the Scheduled Tasks of an Agent

**Tests:** internal/sandbox/schedule_test.go, cmd/spekk/sandbox_help_test.go

## Success Criteria

- `<agent>` is the name of a recorded sandbox that has an agent id, or an agent id (a UUID). Anything else is an error that says which forms are accepted.
- `<schedule>` is a schedule id (a UUID) or a slug. A slug is found by a list of the agent's schedules.
- `spekk schedule list <agent>` prints one row for each schedule, from all pages: id, slug, cron expression, timezone, role, enabled, and next fire time.
- `spekk schedule create <agent> --slug <slug> --cron_expression <expr> --task_text <text> [--timezone <tz>] [--custom_context <text>] [--role <role>] [--enabled true|false]` sends only the given fields and prints the new schedule as JSON on stdout.
- `spekk schedule update <agent> <schedule> [field flags]` sends a `PATCH` with only the given fields, and prints the schedule as JSON. An update with no field is an error. A field flag may take an empty value, so `--custom_context ""` clears it.
- `spekk schedule delete <agent> <schedule>` deletes the schedule.
- `spekk schedule run <agent> <schedule>` triggers the schedule and prints the `message_id`. A `409` prints the `detail` from the control host.
- The flag names are the API field names. `--enabled` takes `true` or `false` only.
- `spekk schedule --help` lists the five subcommands, and `spekk help` lists `schedule`.

## Verification

- `go test ./internal/sandbox` drives each subcommand against an `httptest` control host that follows the contract in the parent spec, including a two-page list, a slug lookup, and a `409` on run.
