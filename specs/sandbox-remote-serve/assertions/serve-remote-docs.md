---
id: serve-remote-docs
parent: sandbox-remote-serve
created: 2026-11-08T00:00:00Z
priority: 3
status: not_started
depends-on: serve-stop-status
branch: feature/sandbox-remote-serve
---

# The CLI reference documents remote serve

## Success Criteria

- `docs/cli-reference.md` has a `spekk sandbox serve` section covering `start`, `connect`, `status`, `stop`, the `--json` and `--raw` flags, and `/handoff`, with one example of each.
- It documents the `spekk serve` WebSocket protocol: the nonce query parameter, the `coach:*` events in each direction, and the `/health` endpoint.
- It lists the known limits from the parent spec: the shared Claude home, the shared subscription rate limit, no local resume, the worker's credentials in the coach's environment, and the handoff trust boundary.
- `spekk sandbox serve --help` and each subcommand's `--help` match the reference.
- All examples use invented sandbox names.
