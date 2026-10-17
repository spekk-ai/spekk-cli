---
id: docs-show-byte-cap
parent: source-document-index
created: 2026-10-17T19:00:00Z
priority: 1
status: not_started
branch: feature/source-document-index
depends-on: docs-show-command
---

# `spekk docs show` Has a Byte Cap

The 200-line cap of `docs show` does not limit bytes: one document line of 2 MB put 2 MB into the context of the agent that read it.

## Success criteria

- One call prints at most 20,000 bytes of document lines (200 lines of 100 bytes). The output stops before the line that goes over the cap, says how many lines it printed, and gives the next `--lines` value, as the 200-line cap does.
- When the first line of the range is longer than the cap, the call prints the start of that line, cut at a character boundary under the cap, with a note that gives the full length of the line in bytes. The next `--lines` value starts at the line after it.
- A range under both caps prints whole, with no note.

**Tests:** `internal/docs/show_test.go`, one case in `TestRenderShowCapsLinesAndCleansText` (a 2 MB line).
