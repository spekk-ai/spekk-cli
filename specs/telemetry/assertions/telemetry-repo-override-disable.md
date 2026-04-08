---
id: telemetry-repo-override-disable
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 2
status: not_started
depends-on: telemetry-config-file
branch: feature/telemetry
---

# Repo Override to Force-Disable Telemetry

## What Must Be True

A repository's `spekk.config.yaml` can force telemetry off for any user operating in that repo, regardless of their global user consent. The override works in the disable direction only — a repo can never force telemetry on for a user who has not consented.

## Success Criteria

- ✅ `spekk.config.yaml` accepts an optional `telemetry:` section:
  ```yaml
  telemetry:
    disabled: true
  ```
- ✅ When `telemetry.disabled: true` is present in the repo config, `telemetry.IsEnabled()` returns `false` regardless of the user's global config
- ✅ Any other value in `telemetry.disabled` (including `false`, omitted, or missing) leaves the user's global preference in effect
- ✅ The loader explicitly rejects `telemetry.enabled` or `telemetry.consented` or any field that would imply repo-level force-on — these are schema errors with a clear message:
  ```
  Error: spekk.config.yaml may only set 'telemetry.disabled: true'.
  Telemetry cannot be enabled at the repo level — it requires user consent.
  ```
- ✅ `spekk telemetry status` shows the effective state AND the source: e.g. `Telemetry: DISABLED (forced off by ./spekk.config.yaml)` when the repo override is active
- ✅ Unit tests cover: repo override disabled with user enabled, repo override omitted with user enabled, repo override set to false with user enabled, invalid `telemetry.enabled` key rejected
- ✅ Integration test: set up a repo with `telemetry.disabled: true`, user globally enabled, run a coach session, verify zero events captured
- ✅ `spekk telemetry enable` refuses to enable when run inside a repo with `telemetry.disabled: true` and prints a clear message explaining why

## Security Rationale

A sensitive repo (customer code, financial data, defense contracts) can ship with `telemetry.disabled: true` committed to git. Every contributor, even one who has enabled telemetry globally, is automatically opted out while working in that repo. This is non-circumventable short of editing the repo config.

The inverse is intentionally impossible: a malicious repo cannot ship with `telemetry.enabled: true` to exfiltrate data from unsuspecting contributors.

## Out of Scope

- Per-directory granularity (repo-level is the finest granularity)
- Time-based overrides
- Partial overrides (e.g., disable only spec deltas)

## Notes

This is a small assertion but an important one. It's the difference between "telemetry you can opt into" and "telemetry you can be forced into." The second kind never ships.
