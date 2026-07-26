---
id: scan-time-suppression
parent: observer-dont-flag
created: 2026-07-26T12:00:00Z
priority: 1
status: not_started
depends-on: dont-flag-file-schema
---

# Matching Drift Is Not Observed at All — Suppression Happens at Scan Time

## Description

The suppression file is consulted before an observation is born. Drift whose
evidence paths or would-be slug match an active entry produces no observation,
no `observer/<slug>` branch, no announcement — it is invisible to the entire
downstream lifecycle.

## Success Criteria

- The observer's scan flow consults `.spekk/dont-flag.yaml` (as it exists on
  main) before creating any observation.
- Drift is suppressed when any active (non-expired) entry matches either:
  - any of the drift's evidence file paths (glob match), or
  - the slug the observation would be given (slug-pattern match)
- Suppressed drift produces **nothing**: no observation file, no branch, no
  index row, no announcement — suppression is not a status, it is
  non-existence.
- Suppression applies only to *new* observations. An existing
  `observer/<slug>` branch created before an entry was added is untouched by
  the suppression file (its lifecycle is governed by the branch state
  machine); adding an entry prevents the *next* birth, it does not resolve
  the current one.
- An expired entry (its `until` date has passed) suppresses nothing: the next
  scan may legitimately flag the drift again.

**Note:** the match target for path globs is the drift's evidence paths (what
would become `affected`), not every file the scan happened to read while
finding it.
