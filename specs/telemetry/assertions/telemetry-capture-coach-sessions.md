---
id: telemetry-capture-coach-sessions
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 1
status: not_started
depends-on: telemetry-event-schema
branch: feature/telemetry
---

# Capture Coach Sessions

## What Must Be True

When telemetry is enabled, the coach agent records every message exchanged during a coach session and produces a `coach-session/v1` event at session end. The event is written to the local queue after redaction. When telemetry is disabled or consent is missing, this code path is a complete no-op.

## Success Criteria

- ✅ New file `internal/telemetry/capture/coach.go` with `CoachSessionRecorder` struct
- ✅ `Recorder.StartSession(installID string) *CoachSession` begins a new session and returns a handle
- ✅ `session.AddMessage(role, content)` appends a message (role is `user` or `coach`)
- ✅ `session.SetOutcome(specsCreated []string, assertionsCreated int, branch string)` records the spec outcome
- ✅ `session.End() (*events.CoachSession, error)` finalizes the session, applies redaction, and writes to the local queue
- ✅ Every capture code path is guarded by `if !telemetry.IsEnabled() { return noop }` — confirmed by unit tests asserting zero queue writes when disabled
- ✅ Coach CLI (`cmd/spekk/coach.go` or equivalent) wires the recorder in: starts on session start, captures each user prompt + coach reply, calls `End()` on session exit
- ✅ The recorder is injected via interface so unit tests can substitute a fake
- ✅ A session that ends with an error (panic, crash) still writes whatever was captured up to that point (defer-based flush)
- ✅ Unit tests cover: capture with telemetry enabled writes one event to queue, capture with telemetry disabled writes zero events, session with zero messages produces no event (filters empty sessions), session with outcome fields populated, session end during a panic still persists
- ✅ Integration test: run a scripted coach session with telemetry enabled, verify a single event file appears in the queue directory, verify the event parses as valid `coach-session/v1`

## Message Content Rules

- **User messages**: captured verbatim BEFORE redaction, then passed through redaction pass
- **Coach messages**: captured verbatim BEFORE redaction (these are model outputs that may contain paths/patterns), then redacted
- Tool calls made by the coach (file reads, bash commands) are **not** captured — only the conversational messages

## Error Handling

- If writing to the queue fails (disk full, permission denied), log a single warning to stderr but do not crash the CLI
- Telemetry failures must never break the user's actual work

## Out of Scope

- Builder session capture (future work)
- Capturing tool calls (intentionally omitted for privacy)
- Streaming/incremental upload during a session (events are written on session end only)

## Notes

The coach session is the primary signal. Everything else in this spec exists to make this one capture safe, private, reviewable, and user-controlled.
