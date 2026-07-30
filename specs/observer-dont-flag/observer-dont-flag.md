---
id: observer-dont-flag
created: 2026-07-26T12:00:00Z
priority: 2
---

# Don't-Flag Suppression File — Human-Gated Scan Exclusions

## Overview

`.spekk/dont-flag.yaml`, committed on main, lists drift the team has decided
the observer should not flag at all. It is consulted at scan time: matching
drift is never observed, never branched, never announced. Because the file
lives on main and changes via PR, every suppression carries a reviewer and a
reason — permanent dismissal of a finding is a small, auditable PR, not a
prompt-side judgment call.

This complements the branch lifecycle (`specs/observation-lifecycle/`):
parking a branch suppresses one specific finding; a dont-flag entry suppresses
a class of drift before an observation is ever born.

## Design

Entry shape:

```yaml
# .spekk/dont-flag.yaml
- match: "internal/legacy/**"          # path glob or observation slug pattern
  reason: "Legacy package scheduled for deletion in Q4; drift is expected."
  by: "william"
  until: 2026-12-31                    # optional; absent = permanent
- match: "parser-drops-*"
  reason: "Known parser looseness, accepted; see ADR-014."
  by: "william"
```

- `match` — required; a path glob (matched against an observation's `affected`
  paths) or a slug pattern (matched against the would-be observation slug)
- `reason` — required
- `by` — required
- `until` — optional date; when present, the entry expires and stops
  suppressing after that date; absent means permanent

## Resolved Questions

- **Expiry surfacing:** silent expiry in v1 — no warnings from the scan or
  `spekk validate` when an entry has expired or is about to. An expired
  entry simply stops suppressing, and the next scan may re-flag the drift.
- **Timezone of `until`:** a date-only value interpreted as end-of-day UTC:
  the entry suppresses through the whole named day and expires at the
  following UTC midnight.

## Assertions

See `assertions/` for what must be true.
