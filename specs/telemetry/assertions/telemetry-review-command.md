---
id: telemetry-review-command
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 2
status: not_started
depends-on: telemetry-local-queue
branch: feature/telemetry
---

# `spekk telemetry review` Command

## What Must Be True

A new CLI subcommand `spekk telemetry review` lists every event currently in the local queue and lets the user view the full contents of any event before it's uploaded. This is the user's primary tool for inspecting what would be sent.

## Success Criteria

- ✅ `spekk telemetry review` subcommand registered
- ✅ Default output (no args): table showing all queued events
  ```
  ID                    SCHEMA              CAPTURED              SIZE
  evt-01H8ZQK...        coach-session/v1    2026-04-08 17:03:11   4.2 KB
  evt-01H8ZR2...        spec-delta/v1       2026-04-08 17:10:54   1.8 KB
  (2 events, 6.0 KB total)
  ```
- ✅ Empty state: prints `Queue is empty. No events to review.`
- ✅ `spekk telemetry review <event-id>` prints the full JSON of that event to stdout
- ✅ `spekk telemetry review <event-id> --pretty` prints indented JSON with syntax highlighting if a TTY is detected
- ✅ `spekk telemetry review --all` prints every event in the queue to stdout, separated by `---`
- ✅ `spekk telemetry review --json` prints the index as JSON (not human table) for scripting
- ✅ Exit code 0 if queue has events or is empty, non-zero only on errors (file read failure, etc.)
- ✅ Integration test: enqueue 3 events, run `review`, verify table shows all 3 with correct sizes
- ✅ Integration test: enqueue 1 event, run `review <id>`, verify full JSON printed to stdout
- ✅ Integration test: run `review non-existent-id`, verify non-zero exit and clear error message

## Design Principles

- **Everything visible.** The whole point is user visibility. No abbreviated output by default for full-event mode.
- **Zero side effects.** Running `review` must never modify, delete, or upload anything. Pure read.
- **Scriptable.** `--json` mode exists so users can pipe through `jq` or their own tooling.

## Out of Scope

- Interactive TUI to browse events (too much scope; `review` + `review <id>` is enough)
- Filtering by schema, date, or install ID (can be added later if needed)
- Diffing between events

## Notes

The consent flow promises users they can review every byte before it's sent. This command is how that promise is kept. If `review` doesn't work or is hard to use, the whole opt-in model loses credibility.
