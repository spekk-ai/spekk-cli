---
id: stale-lock-is-reported
parent: lock-is-a-live-claim
created: 2026-08-21T17:35:19Z
priority: 2
status: not_started
---

# `spekk validate` Reports a Stale Lock on an `in_progress` Assertion

A lock is a live claim, so an old one is almost always dead. `validate` reports an `in_progress` assertion whose `locked-by` is stale. The old invariant made this shape legal and therefore unreportable.

## Success criteria

- `validate` uses `parser.IsLockStale` for the judgement. It does not parse the timestamp itself and it does not define its own threshold.
- An `in_progress` assertion with a non-empty `locked-by` for which `IsLockStale` returns true is reported, one line per assertion, on stderr:

  ```
  Warning: specs/foo/assertions/bar.md: locked-by "builder-host-1750000000" is stale; no builder holds this assertion
  ```

- An assertion with an **empty** `locked-by` is not reported. `IsLockStale("")` returns true, but an absent lock is the legal unlocked state from `unlocked-in-progress-is-legal`, not a stale one. Guard on the non-empty value before the call.
- A `locked-by` whose tail is not a unix timestamp is reported as stale. `IsLockStale` already returns true for a malformed value, and a lock nobody can date is a lock nobody can trust. This is what catches a value a coach invented to satisfy the old rule.
- The report is a **warning, not a failure**. `validate` exits `0` when the only findings are stale locks, and its stdout still holds only the `validate: N specs, M assertions OK` line.
- Lines are sorted by file path, matching the deterministic ordering of the failure report.

**Note:** Warnings go to stderr and failures to stdout. That split is the existing contract in `validate-command`: stdout stays clean and diffable for CI, so a warning must not enter it.

**Tests:** `internal/validate/` — an `in_progress` assertion with a lock timestamped over two hours ago is reported; one with a fresh timestamp is not; one with an empty `locked-by` is not; one with a non-numeric tail (`coach-invented-value`) is reported; the exit code stays `0` and stdout stays the summary line in every case.
