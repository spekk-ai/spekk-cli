---
id: machine-mode-docs
parent: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 2
status: not_started
branch: feature/sandbox-machine-mode
depends-on: machine-destroy
---

# The CLI Reference Describes Machine Mode

## Success Criteria

- `docs/cli-reference.md` has a section on machine mode: the two commands, the stdin JSON of each, the stage lines, the exit status, the labels, and the rule that no secret is in a flag, an environment variable, or stdout.
- `spekk sandbox create --help` and `spekk sandbox destroy --help` list `--machine`.
- `docs/advanced/sandbox-architecture.md` names the `claude_auth` heartbeat field and protocol `1.2`.
- No doc names a private host, a customer, or an internal URL.

## Verification

- Read the files and the help output.
