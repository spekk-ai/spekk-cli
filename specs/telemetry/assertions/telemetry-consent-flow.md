---
id: telemetry-consent-flow
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 1
status: not_started
depends-on: telemetry-config-file
branch: feature/telemetry
---

# Telemetry Consent Flow

## What Must Be True

`spekk telemetry enable` walks the user through an interactive consent screen that clearly discloses what is captured, why, where it goes, and how to opt out. The user must explicitly accept before any telemetry is recorded.

## Success Criteria

- ✅ `spekk telemetry enable` subcommand registered in the CLI
- ✅ On invocation, prints a multi-section consent screen:
  ```
  Spekk Telemetry

  WHAT IS CAPTURED (only when enabled):
    - Your conversations with the coach agent (messages sent + coach replies)
    - Summary of specs created or modified during coach sessions
    - Git diffs of subsequent spec changes (the "poke hole" deltas)

  WHAT IS NEVER CAPTURED:
    - File contents outside spec files
    - Source code
    - Environment variables or secrets
    - Any non-coach CLI interactions (builder sessions, spekk next, etc.)
    - Your name, email, or any identifier you haven't opted in to share

  WHY WE COLLECT THIS:
    To improve the coach agent by learning where PM-authored specs fall
    short and where engineering critique catches gaps the coach should
    have caught. The data directly shapes future coach behavior.

  WHERE IT GOES:
    https://telemetry.spekk.ai/v1/events  (or a custom endpoint if configured)

  HOW TO CONTROL IT:
    spekk telemetry review    - see every queued event before it's sent
    spekk telemetry flush     - upload queued events (manual only)
    spekk telemetry disable   - stop all future capture immediately
    spekk telemetry purge     - delete all local events
    spekk telemetry delete <id> - delete a specific event

  This is OPT-IN. Nothing is sent until you run `spekk telemetry flush`.

  To enable, type the word "enable" and press Enter.
  To cancel, press Ctrl-C or type anything else.
  >
  ```
- ✅ User must type the literal word `enable` to opt in — any other response aborts
- ✅ On `enable`: creates `~/.config/spekk/telemetry.yaml` with `enabled: true`, generates `install_id`, sets `consented_at` to current UTC, sets `consent_version: 1`
- ✅ Prints confirmation: `Telemetry enabled. Install ID: anon-9f3e...  To disable anytime: spekk telemetry disable`
- ✅ On cancel: prints `Telemetry not enabled. You can opt in anytime with 'spekk telemetry enable'.` and exits 0
- ✅ `spekk telemetry disable` subcommand sets `enabled: false` in the config, prints `Telemetry disabled. All future capture and upload stopped. Queued events remain locally — purge with 'spekk telemetry purge'.`
- ✅ `spekk telemetry status` prints current state: enabled/disabled, install ID (if any), endpoint, queue size, last flush time
- ✅ Integration test: runs `enable` with piped stdin `enable\n`, verifies config file created
- ✅ Integration test: runs `enable` with piped stdin `yes\n`, verifies config file NOT created (must be literal word `enable`)
- ✅ Integration test: `status` on a fresh install prints `disabled`

## Consent Versioning

The `consent_version: 1` field exists so future changes to what we capture require re-consent. If `consent_version` in the code exceeds the version in the user's config, treat as disabled until they re-consent via `spekk telemetry enable`.

## Out of Scope

- Remote-fetched consent text (it's hardcoded in the binary)
- Multi-language consent screens (English only for MVP)
- GUI consent dialog

## Notes

The "literal word enable" requirement is intentional friction. `y`/`yes` is too easy to type accidentally. Typing `enable` is a conscious action.
