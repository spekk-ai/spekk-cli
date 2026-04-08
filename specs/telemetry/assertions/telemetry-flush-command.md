---
id: telemetry-flush-command
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 2
status: not_started
depends-on: telemetry-upload-endpoint
branch: feature/telemetry
---

# `spekk telemetry flush` Command

## What Must Be True

A new CLI subcommand `spekk telemetry flush` uploads all queued events to the configured endpoint. It is the only way to trigger an upload for the MVP — there is no background auto-flush.

## Success Criteria

- ✅ `spekk telemetry flush` subcommand registered
- ✅ Reads the telemetry config and exits with a clear error if telemetry is not enabled:
  ```
  Telemetry is not enabled. Run `spekk telemetry enable` first.
  ```
- ✅ Loads queue and prints a summary before uploading:
  ```
  Found 12 events (48.3 KB) queued for upload to https://telemetry.spekk.ai/v1/events
  ```
- ✅ `--dry-run` flag shows what would be uploaded without actually making HTTP calls
- ✅ Default behavior: upload immediately without additional confirmation (the user already opted in)
- ✅ `--confirm` flag adds an interactive `[y/N]` prompt before uploading
- ✅ Progress output during upload: `Uploading batch 1/3...` (when multiple batches are needed)
- ✅ On success: deletes uploaded events from the queue, prints `Uploaded 12 events. Queue now empty.`
- ✅ On partial success (some events rejected): prints per-event outcome and retains rejected events
- ✅ On transient failure: prints error, retains all events in queue, exit code 2
- ✅ On permanent failure (4xx): prints error, drops offending events, exit code 1
- ✅ Empty queue: prints `No events to upload. Queue is empty.` and exits 0
- ✅ Integration test against fake HTTP server: enqueue events, run flush, verify HTTP POST, verify queue emptied
- ✅ Integration test: flush with no events produces empty-queue message
- ✅ Integration test: flush with telemetry disabled exits with clear error

## Exit Codes

- `0` — success (including empty queue)
- `1` — permanent failure (config error, events dropped due to 4xx)
- `2` — transient failure (network down, 5xx, events retained for next flush)

## Out of Scope

- Background auto-flush (future work)
- Flush scheduling (cron-like)
- Selective flush by event ID or schema

## Notes

Flush is a deliberate action. Users ran `enable`, they reviewed with `review`, now they run `flush` to upload. Three explicit steps, each reversible. That's the trust story.
