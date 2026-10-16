---
id: docs-and-help
parent: sandbox-kind
created: 2026-10-15T23:40:00Z
priority: 2
status: not_started
depends-on: list-expired-and-all-hosts
branch: sandbox-kind
---

# The Docs and the Help Describe Sandbox Kinds

**Tests:** cmd/spekk/sandbox_help_test.go

## Success Criteria

- `docs/advanced/sandbox-architecture.md` has a section that describes the kinds, the four registry fields, the labels for each provider, the marker file, the `list` flags, and that expiry does not destroy a sandbox. It says that each new DigitalOcean tag is an account-level tag that stays after a destroy.
- `spekk sandbox create --help` shows its title one time, and lists `--one-off`, `--expires` and `--purpose` with their defaults.
- `spekk sandbox list --help` exists and lists `--expired` and `--all-hosts`.
- `docs/cli-reference.md` describes the new `create` and `list` flags.

## Verification

- A test in `cmd/spekk` checks that the create help names `--one-off`, `--expires` and `--purpose`, and that the list help names `--expired` and `--all-hosts`.
- `go build ./cmd/spekk && ./spekk sandbox create --help && ./spekk sandbox list --help` show the flags.
