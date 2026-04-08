---
id: telemetry
created: 2026-04-08T17:00:00Z
priority: 2
---

# Telemetry

## Overview

Add opt-in, local-first telemetry to the spekk CLI that captures coach agent conversations and spec evolution. The data is used to improve the coach agent — specifically to learn where PM-authored specs fall short and where engineering "pokes holes" that the coach should have caught up front.

## Motivation

The coach agent today has no feedback loop. It proposes specs, users accept them, and what happens next — engineering critique, spec edits, implementation failures — never makes it back to inform the coach. The richest training signal in spec-driven development is the delta between **"what the PM thought they wanted"** and **"what engineering actually built after poking holes."** Capturing that delta lets us improve the coach agent to spot gaps earlier.

Specifically, we want to learn:

- What questions should the coach ask that it currently doesn't?
- What spec shapes consistently get amended by engineering before implementation?
- Which assertions are most likely to be marked `failed` and why?
- Where does the imperative-to-declarative translation break down?

## Non-Goals

This is **not** general analytics. We are not collecting:

- CLI command usage counts
- Performance metrics
- Crash reports (those belong in a separate error-reporting system)
- Any data from non-coach interactions (for the MVP)

We are not building:

- A dashboard
- A real-time observability platform
- User tracking across sessions

## Core Contract (non-negotiable)

1. **Default off.** No data leaves the machine unless the user runs `spekk telemetry enable` and accepts the consent screen.
2. **Local-first.** All captured events are written to `~/.spekk/telemetry/queue/` as JSON files. Nothing is sent until the user (or opt-in auto-flush) triggers an upload.
3. **Reviewable.** `spekk telemetry review` displays every queued event in full so the user can see exactly what would be sent.
4. **Deletable.** Users can delete individual queued events, purge the entire queue, or request server-side deletion.
5. **Redacted by default.** File contents are never captured (only paths). Common secret patterns are stripped. Users can add additional deny-patterns.
6. **User-global config.** Telemetry consent is a user decision, not a repository property. Config lives at `~/.config/spekk/telemetry.yaml`.
7. **Repo can force-off but never force-on.** A repo's `spekk.config.yaml` may disable telemetry for sensitive projects, but cannot enable telemetry for a user who has not consented.
8. **Anonymous by default.** Events include a random install ID, not user identity. Email attach is opt-in only.

## Design

### Data flow

```
Coach session ─┐
               ├──> Local capture ──> Redaction ──> ~/.spekk/telemetry/queue/{event-id}.json
Spec delta ────┘                                                │
                                                                │  user runs
                                                                │  spekk telemetry flush
                                                                ▼
                                                    Upload endpoint (configurable)
                                                                │
                                                                ▼
                                                         Training corpus
```

### What gets captured (MVP)

**Coach session event:**
```json
{
  "schema": "coach-session/v1",
  "id": "evt-01H...",
  "install_id": "anon-9f3e...",
  "captured_at": "2026-04-08T17:00:00Z",
  "session": {
    "started_at": "...",
    "ended_at": "...",
    "messages": [
      {"role": "user", "content": "..."},
      {"role": "coach", "content": "..."}
    ]
  },
  "spec_outcome": {
    "specs_created": ["external-lock-backend"],
    "assertions_created": 9,
    "branch": "feature/external-lock-backend"
  },
  "redacted": true,
  "redaction_rules_applied": ["file-contents", "env-vars", "api-keys"]
}
```

**Spec delta event** (captured on subsequent commits that touch specs created by a coach session):
```json
{
  "schema": "spec-delta/v1",
  "id": "evt-01H...",
  "install_id": "anon-9f3e...",
  "captured_at": "2026-04-09T09:00:00Z",
  "original_session_id": "evt-earlier...",
  "spec_id": "external-lock-backend",
  "delta_summary": {
    "assertions_added": 1,
    "assertions_removed": 0,
    "assertions_modified": 2
  },
  "git_diff": "...(redacted)..."
}
```

### Config shape

```yaml
# ~/.config/spekk/telemetry.yaml
enabled: true
install_id: anon-9f3e8b2c...
consented_at: 2026-04-08T17:00:00Z
consent_version: 1
endpoint: https://telemetry.spekk.ai/v1/events
capture:
  coach_sessions: true
  spec_deltas: true
redaction:
  extra_patterns:
    - "ACME_SECRET_.*"
    - "/Users/.*/private/.*"
email: ""  # opt-in, empty by default
```

### Repo-level override

```yaml
# spekk.config.yaml (in the repo root)
telemetry:
  disabled: true  # force-off, overrides user preference
```

No `telemetry: enabled: true` — you cannot force-on.

## Success Criteria

- ✅ A brand-new install never sends telemetry under any circumstances without an explicit `spekk telemetry enable` and consent
- ✅ Users can see every byte that would be sent before it goes anywhere (`spekk telemetry review`)
- ✅ Users can delete past events before they're uploaded
- ✅ File contents never appear in captured events — only paths
- ✅ Sensitive repos can force-disable telemetry via `spekk.config.yaml`
- ✅ Coach sessions captured include enough signal (messages + spec outcomes) to learn PM-vs-engineering divergence
- ✅ All telemetry code paths are covered by tests, including "telemetry enabled but network down" scenarios

## Out of Scope (Future Work)

- Builder conversation capture
- CLI command usage telemetry
- Crash/error reporting
- Automatic background flushing
- Real-time streaming
- Multi-user install identification (organizations, teams)
- User-hosted telemetry collectors beyond "set a custom endpoint URL"

## Assertions

See `assertions/` subfolder. Work on this spec lives on branch `feature/telemetry`.
