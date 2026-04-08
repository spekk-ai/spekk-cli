---
id: telemetry-local-queue
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 1
status: not_started
depends-on: telemetry-event-schema
branch: feature/telemetry
---

# Telemetry Local Queue

## What Must Be True

A local queue stores captured events as individual JSON files in `~/.spekk/telemetry/queue/`. The queue supports append, read-all, delete-one, and purge operations. Events stay in the queue until uploaded or explicitly deleted.

## Success Criteria

- ✅ New package `internal/telemetry/queue/`
- ✅ Queue directory is `$HOME/.spekk/telemetry/queue/` (not XDG — this is runtime state, not config)
- ✅ Queue directory and parent dirs created with mode `0700` on first write
- ✅ `Enqueue(event []byte, id string) error` writes the event to `{id}.json` with mode `0600`
- ✅ `List() ([]QueuedEvent, error)` returns all queued events sorted by `captured_at` ascending
- ✅ `Read(id string) ([]byte, error)` returns the raw event bytes
- ✅ `Delete(id string) error` removes a single event; missing id is not an error (idempotent)
- ✅ `Purge() error` removes all files in the queue directory (not the directory itself)
- ✅ `Size() (count int, bytes int64, err error)` returns queue statistics for `spekk telemetry status`
- ✅ Concurrent `Enqueue` from multiple goroutines produces distinct files (use unique IDs, not a sequence counter)
- ✅ The queue reads event metadata (schema, id, captured_at) from the file itself — no separate index file
- ✅ Unit tests cover: enqueue one, enqueue many, list ordering, delete one, delete missing, purge, permissions, concurrent enqueue, malformed file in queue (warns but doesn't crash)
- ✅ Integration test: fill the queue with 10 events, list them, delete every other one, verify remaining 5

## File Naming

- `{event-id}.json` where event-id is the event's `id` field (ULID or UUID)
- No timestamps in filenames (timestamps live inside the event); allows renaming/moving without corrupting data

## Crash Safety

- Enqueue writes to a temp file (`{id}.json.tmp`), then renames to `{id}.json` atomically
- Partial writes never leave corrupted `.json` files in the queue
- On startup, queue package scans for `.tmp` files and removes them (they're always partial writes)

## Out of Scope

- Size limits / rotation (for MVP, unbounded queue — add limits in future work)
- Encryption at rest (events are already redacted; filesystem permissions are sufficient)
- Remote storage as a queue backend (local only for MVP)

## Notes

The queue is deliberately dumb. One file per event, no database, no index. Human-inspectable with `ls ~/.spekk/telemetry/queue/ && cat ~/.spekk/telemetry/queue/*.json`.
