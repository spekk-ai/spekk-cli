---
id: conversation-open-cli
parent: sandbox-conversation-open
created: 2026-07-23T00:00:00Z
priority: 1
status: done
depends-on: conversation-open-contract
---

# Agent Trigger: `spekk conversation open`

The Claude process inside a sandbox session can request a conversation by
running a `spekk` subcommand as a tool. The subcommand writes one request into
the per-session spool directory and exits; it never touches the WebSocket
connection and never supplies a session id.

## Success Criteria

- `spekk conversation open` is registered as a subcommand of the `spekk` CLI
  (`cmd/spekk`), dispatched the same way as existing subcommands. `spekk
  conversation --help` and `spekk conversation open --help` print usage.
- Flags:
  - `--title <text>` — required.
  - `--body <text>` — required.
  - `--severity <info|warning|critical>` — optional, defaults to `info`.
- The command discovers the spool directory from the environment variable the
  worker sets on the session, reading its name from the shared `conversation`
  package's env-var constant (`SPEKK_CONVERSATION_SPOOL`) rather than a local
  literal. It writes exactly one JSON file into that directory using the shared
  package's request struct, containing `title`, `body`, and `severity` — and
  **no** `session_id`. On success it exits 0. **Note:** severity validation uses
  the shared package's constants/validity check, so the CLI and the worker agree
  on the allowed values (see `conversation-open-contract`).
- The request file is written atomically: written to a temporary name and
  renamed into place, so a concurrent worker drain never observes a partial
  file. **Note:** distinct invocations must not collide on a filename (e.g. use
  a unique/random component), so two quick calls both survive.
- Missing `--title` or `--body` exits non-zero with a clear message naming the
  missing flag. A `--severity` outside the three allowed values exits non-zero
  with a message listing the valid values (it is rejected here, not silently
  corrected).
- When the spool environment variable is unset or empty, the command exits
  non-zero with a clear message that it must be run inside a sandbox session —
  it does not panic and does not create files in an arbitrary location.
- A test exercises the core: given a spool directory, a valid invocation writes
  one well-formed request file (with no `session_id` key); a missing required
  flag and an invalid severity each return a non-zero result with a legible
  message; and an unset spool variable returns a non-zero result.

**Tests:** cmd/spekk/conversation_test.go
