---
id: telemetry-defaults-to-off
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 1
status: not_started
branch: feature/telemetry
---

# Telemetry Defaults To Off

## What Must Be True

A fresh install of spekk makes zero network calls related to telemetry under any circumstances. Telemetry is opt-in only, requiring explicit `spekk telemetry enable` followed by consent.

## Success Criteria

- ✅ A brand-new install (no `~/.config/spekk/telemetry.yaml` file) produces zero telemetry network activity during any CLI invocation
- ✅ Running `spekk next`, `spekk show`, `spekk coach`, `spekk builder`, or any other command on a fresh install never attempts a telemetry upload
- ✅ Coach and builder agents may capture events to a local disk queue only if telemetry is explicitly enabled — never otherwise
- ✅ An integration test starts with `HOME=$(mktemp -d)` and asserts zero outbound HTTP requests to any telemetry endpoint across a representative session
- ✅ Removing `~/.config/spekk/telemetry.yaml` (disabling telemetry) immediately stops all future capture and all future uploads
- ✅ A unit test verifies the telemetry package's public functions are all no-ops when `enabled: false` or the config file is missing
- ✅ Uninstalling spekk removes or preserves the local queue per user expectation (document the decision in the spec — recommend: preserve, so reinstall can review or purge)

## Defensive Checks

- The telemetry package must refuse to send under any of the following conditions, even if called directly:
  - Config file missing
  - Config file present but `enabled: false`
  - Config file present but `consented_at` missing
  - Repo-level `spekk.config.yaml` sets `telemetry.disabled: true`
- Each of the above must have a test covering both "no events captured" and "captured events not uploaded."

## Out of Scope

- What the opt-in flow looks like (separate assertion)
- How capture actually works once enabled (separate assertions)

## Notes

This is the contract assertion. If this invariant ever breaks, the project loses user trust permanently. Every other telemetry assertion must satisfy this rule.
