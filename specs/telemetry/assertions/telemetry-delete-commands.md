---
id: telemetry-delete-commands
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 2
status: not_started
depends-on: telemetry-local-queue
branch: feature/telemetry
---

# Telemetry Delete Commands

## What Must Be True

Users can delete individual queued telemetry events or purge the entire queue. Both operations are local-only — they do not reach out to the server (server-side deletion is a separate concern handled by the privacy policy / email process).

## Success Criteria

- ✅ `spekk telemetry delete <event-id>` subcommand registered
- ✅ On success: prints `Deleted event {id}. Queue now has N events.`
- ✅ Missing event ID: prints `No event with ID: {id}` and exits non-zero
- ✅ `spekk telemetry purge` subcommand registered
- ✅ Purge prints the current queue size and requires confirmation:
  ```
  This will delete all 12 queued events (48.3 KB) from your local queue.
  Events already uploaded are NOT affected.
  To request server-side deletion, email privacy@spekk.ai with your
  install ID: anon-9f3e8b2c...

  Type 'purge' to confirm, anything else to cancel.
  >
  ```
- ✅ Requires typing the literal word `purge` to proceed (same friction pattern as `enable`)
- ✅ `spekk telemetry purge --yes` skips the confirmation
- ✅ On success: prints `Purged 12 events. Queue is now empty.`
- ✅ Empty queue: prints `Queue is already empty.` and exits 0 (no-op, no error)
- ✅ Integration tests: delete one of many, delete missing, purge full queue, purge empty queue, purge with `--yes` flag

## Privacy Policy Reference

The purge confirmation prompt explicitly tells the user how to request server-side deletion for events they've already uploaded. This is a legal/trust requirement — users must have a clear path to "forget me" even after flush.

## Out of Scope

- Automated server-side deletion requests from the CLI (for MVP, use email)
- Filtering deletes by schema or date
- Undo for deletes (deletes are immediate and permanent locally)

## Notes

These commands close the loop on local control. Between `enable`/`disable`, `review`, `flush`, `delete`, and `purge`, the user has complete visibility and control over the local queue.
