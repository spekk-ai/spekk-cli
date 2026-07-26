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

## Open Questions

- **Expiry surfacing:** should the scan (or `spekk validate`) warn when an
  entry has expired or is about to, so stale suppressions get cleaned up, or
  is silent expiry enough? Left to the builder; silent expiry is the minimum.
- **Timezone of `until`:** proposed as a date-only value interpreted as
  end-of-day UTC; confirm before implementing.

## Assertions

See `assertions/` for what must be true.
