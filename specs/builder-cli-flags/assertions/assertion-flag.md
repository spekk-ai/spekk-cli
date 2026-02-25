---
id: assertion-flag
parent: builder-cli-flags
created: 2026-02-21T12:00:00Z
priority: 1
status: done
---

# --assertion Flag Targets Specific Assertion

## Description

The `--assertion <id>` flag tells the builder to work on exactly that assertion, bypassing the priority queue.

## Success Criteria

- `spekk builder --assertion login-button` builds only that assertion
- Ignores priority ordering - builds it regardless of priority
- Works even if assertion status is `done` (allows re-running)
- Shows error if assertion ID doesn't exist
- Exits after building (single assertion, no loop)
