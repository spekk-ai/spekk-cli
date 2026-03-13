---
id: sandbox-command-routing
parent: sandbox-command
created: 2026-03-12T15:00:00Z
priority: 1
status: done
branch: feature/sandbox-command
---

# Sandbox Command Routing

## Requirement

`spekk sandbox <subcommand>` is a top-level CLI command registered in `bin/spekk.js`. It routes to subcommands: `create`, `list`, `status`, `ssh`, `destroy`, `deploy`. The implementation lives in `src/sandbox/`.

## Success Criteria

- `spekk sandbox --help` prints usage showing all six subcommands with descriptions
- `spekk sandbox` with no subcommand prints help text and exits with code 0
- `spekk sandbox unknown-cmd` prints an error and exits with code 1
- `bin/spekk.js` has a `sandbox` case in its command switch that delegates to `src/sandbox/cli.js`
- `src/sandbox/cli.js` exports a `launchSandbox(args)` function that parses subcommands and flags
- `spekk sandbox create` accepts `--name` (required), `--region` (default `nyc1`), `--size` (default `s-2vcpu-4gb`)
