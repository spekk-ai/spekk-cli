---
slug: turn-keepalive-reserves-a-taken-protocol-version
type: outdated_specs
severity: medium
status: open
created: 2026-10-21T00:00:00Z
affected:
  - specs/turn-progress-reporting/assertions/turn-keepalive-frame-sent-during-tool-calls.md
  - specs/turn-progress-reporting/assertions/sandbox-docs-record-turn-progress-protocol.md
  - specs/sandbox-sign-in-terminal/assertions/agent-sign-in-terminal.md
  - cmd/sandbox/protocol.go
  - cmd/sandbox/protocol_test.go
  - docs/advanced/sandbox-architecture.md
---

# The Queued Keepalive Assertion Still Reserves Protocol `1.3`, Which Is Now Taken

## Issue Description

The `turn-keepalive-frame-sent-during-tool-calls` assertion (`not_started`, so
it is queued for the builder) says the keepalive frame "comes with protocol
version `1.3`", and that `ProtocolVersion` and "the pinning test" read `"1.3"`.
Neither holds any more. Since that assertion was written, three additive minor
versions have shipped:

- `1.3`: the sign-in terminal frames (#289)
- `1.4`: optional channel routing on `conversation_open` (df08e89)
- `1.5`: the optional `repository` object on a `message` frame (#294)

df08e89 also deleted `TestProtocolVersionPinned`, so there is no pinning test
to update.

The cause is that several specs state a literal `ProtocolVersion` value rather
than a rule. Each additive bump makes one of them false. Two `done` assertions
have the same problem:

- `agent-sign-in-terminal` still requires "`ProtocolVersion` is `1.3`".
- `sandbox-docs-record-turn-progress-protocol` requires the doc to say the
  keepalive is "reserved for version `1.3`", and its done note says the
  keepalive obligation "comes back with `1.3`".

## Evidence

- `cmd/sandbox/protocol.go:17`: `const ProtocolVersion = "1.5"` (set by 8f602d8, #294).
- `cmd/sandbox/protocol_test.go`: `TestProtocolVersionPinned` was removed in df08e89.
- `docs/advanced/sandbox-architecture.md:61-65`: lists `1.3` as the sign-in
  terminal, `1.4` as the conversation channel, and `1.5` as the repository object.
- `specs/turn-progress-reporting/assertions/turn-keepalive-frame-sent-during-tool-calls.md:28`:
  "The frame comes with protocol version `1.3`. `ProtocolVersion` … and the pinning test read `"1.3"`".
- `specs/turn-progress-reporting/assertions/sandbox-docs-record-turn-progress-protocol.md:19,24,27`:
  "reserved for version `1.3`" and "comes with `1.3`".
- `specs/sandbox-sign-in-terminal/assertions/agent-sign-in-terminal.md:20`: "`ProtocolVersion` is `1.3`."

## Impact

When the control host accepts `turn_keepalive`, a builder that follows the
assertion as written would set `ProtocolVersion` back from `1.5` to `1.3` and
restore a pinning test. `1.3` would then mean two different things on the
wire, and a control host could not tell from the version whether an agent
sends keepalives. That is exactly what the assertion's own reasoning says it
wants to avoid. Until then, a reviewer checking the `done` assertions against
the code finds two that the code no longer meets.

## Recommendation

State the rule instead of a number. The keepalive takes the next minor
version after the current `ProtocolVersion` when it ships. Assertions record
which version a feature introduced, not what the constant "is". The remedy
commit on this branch makes these edits to the three assertions. It does not
change any code or status.
