---
id: telemetry-event-schema
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 1
status: not_started
branch: feature/telemetry
---

# Telemetry Event Schema

## What Must Be True

A versioned, documented JSON schema exists for all telemetry event types. Every captured event conforms to one of the defined schemas, and the schema name + version is embedded in every event so the server can evolve compatibly.

## Success Criteria

- ✅ New file `internal/telemetry/events/events.go` defines Go structs for each event type
- ✅ New file `docs/telemetry-schema.md` documents the schemas in human-readable form for the privacy disclosure
- ✅ Every event includes these required envelope fields:
  - `schema` — string, e.g. `coach-session/v1`
  - `id` — ULID or UUID, unique per event
  - `install_id` — from config
  - `captured_at` — RFC 3339 timestamp
  - `redacted` — bool, always `true` after redaction pass
  - `redaction_rules_applied` — list of rule names that matched
- ✅ `CoachSession` event struct includes:
  - `session.started_at`, `session.ended_at`
  - `session.messages` — list of `{role, content}` (role = `user` or `coach`)
  - `spec_outcome.specs_created` — list of spec IDs
  - `spec_outcome.assertions_created` — int
  - `spec_outcome.branch` — string
- ✅ `SpecDelta` event struct includes:
  - `original_session_id` — references the earlier coach session event (optional, may be empty)
  - `spec_id` — string
  - `delta_summary.assertions_added`, `assertions_removed`, `assertions_modified` — ints
  - `git_diff` — string (post-redaction)
- ✅ Each event type has a `ToJSON()` helper that serializes with stable field ordering
- ✅ Each event type has a `Validate()` helper that checks required fields are populated
- ✅ Unit tests cover: valid event serialization, missing required field validation failure, schema version embedded correctly, round-trip JSON encode/decode
- ✅ Schema version is a compile-time constant; bumping it in code triggers a re-consent requirement (see consent-flow assertion)

## Versioning Rules

- `schema: "coach-session/v1"` — `v1` is the current schema version for this event type
- Breaking changes (removing fields, changing types, changing semantics) require a new version: `coach-session/v2`
- Additive changes (new optional fields) stay at `v1`
- When a new version is introduced, bump the overall telemetry `consent_version` so users re-consent

## Out of Scope

- Server-side schema storage or migration
- Builder session events (future work)
- Binary event formats (JSON only)

## Notes

Schema design is a one-way door. Once users start uploading events, changing the shape without a version bump breaks server-side parsing. Err toward more envelope fields now (extensibility) than adding them later.
