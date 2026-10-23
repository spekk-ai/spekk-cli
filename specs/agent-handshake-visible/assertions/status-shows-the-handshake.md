---
id: status-shows-the-handshake
parent: agent-handshake-visible
created: 2026-10-23T06:00:00Z
priority: 1
status: done
depends-on: agent-records-its-handshake
---

# `spekk sandbox status` shows the last handshake and warns about a refused agent

**Tests:** `internal/agentconn/agentconn_test.go`

## Success criteria

- `spekk sandbox status <name>` reads the record over SSH, after the `Agent service` line, and prints two lines:
  - `Agent last connected: <time>`, or `never`.
  - `Agent last handshake failure: <time> (HTTP <code>)`, or `<time> (<error>)` when the failure has no status code, or `none`.
- When the last failure is later than the last success, or the agent never connected and has a failure, the failure line ends with `  WARNING: the agent is not connected to the control host`.
- When the file is not on the machine, the lines say `unknown (no record; the agent-client may be older than this check)`. When SSH fails, they say `unknown`, as the other SSH checks do. When the record is not valid JSON, they say `unknown (the record is not valid)`.
- The agent client and the CLI use one definition of the record and of its path.

## Test

A table test of the function that turns the SSH output into the two lines: a healthy agent, an agent that never connected and is refused with 403, an agent that connected and then failed with no status code, an agent that failed and then connected again, no record, an SSH failure, and a record that is not valid.
