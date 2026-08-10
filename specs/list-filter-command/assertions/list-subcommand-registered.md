---
id: list-subcommand-registered
parent: list-filter-command
created: 2026-07-12T20:30:00Z
priority: 1
status: done
---

# `spekk list` Is a Registered Subcommand

## Description

`spekk list` must be recognized by the CLI dispatcher and appear in the help text.

## Success Criteria

- The `switch command` block in `cmd/spekk/main.go` has a `case "list":` branch
  that calls a `runList(args[1:])` function (or equivalent).
- `spekk help` (the `helpText` constant) includes `list` in the COMMANDS table.
- `spekk list --help` prints a usage line and description of accepted flags
  (`--status`, `--assertions-only`, `--specs-dir`).
- `spekk list` with no flags exits 0 and produces valid JSON output.
- Running `spekk list` on an empty specs directory outputs the same "no specs"
  response as `spekk next --all`.
