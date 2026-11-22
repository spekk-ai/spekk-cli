---
slug: protocol-additions-without-minor-bump
type: underspecified_spec
severity: medium
status: resolved
created: 2026-11-22T00:00:00Z
announced: 2026-11-22T20:03:44Z
affected:
  - specs/sandbox-protocol-version/sandbox-protocol-version.md
  - cmd/sandbox/protocol.go
  - cmd/sandbox/message.go
  - cmd/sandbox/conn_holder.go
  - docs/advanced/sandbox-architecture.md
  - docs/release-notes/RELEASE-NOTES-1.51.0.md
---

# Protocol Additions Gated by a Welcome Capability Shipped Without a Minor Bump

## Issue Description

The sandbox protocol spec says "An additive change bumps the minor" and lists
"Feature-flag negotiation per capability" under **Rejected**. Two recent PRs
added to the WebSocket contract without a bump, and gated the additions on
`welcome` capabilities instead:

- #360 (9a62075) added the `agent_conversation_delivery` welcome capability.
- #361 (d403cc0) added the `agent_current_thread_delivery` welcome capability
  and a new `agent_turn_id` field on the inbound `message` frame.

`ProtocolVersion` is still `"1.6"`, as set by #319 on 2026-10-24, and the
1.51.0 release note says "The WebSocket protocol version stays `1.6`".

The history has both readings of the rule. Version `1.4` added
`conversation_open_channel` as a welcome capability **and** bumped the minor.
The 1.51 additions added capabilities and a frame field **without** a bump.
Both satisfy someone's reading of the spec: either every addition bumps the
minor, or an addition behind a capability needs no bump. The spec does not
say which, and its Rejected list says capability negotiation should not
exist at all.

## Evidence

- `specs/sandbox-protocol-version/sandbox-protocol-version.md:17`: "An additive change bumps the minor." Before the remedy commit, line 22 rejected "Feature-flag negotiation per capability." The remedy moves that item to line 24 and rewords it.
- `cmd/sandbox/protocol.go:17`: `const ProtocolVersion = "1.6"`; the comment
  above it repeats the bump rule.
- `cmd/sandbox/message.go:9`: `AgentTurnID string \`json:"agent_turn_id,omitempty"\``,
  added by d403cc0.
- `cmd/sandbox/conn_holder.go:93-105`: `setCapabilities` accepts
  `conversation_open_channel`, `agent_conversation_delivery`, and
  `agent_current_thread_delivery`.
- `docs/advanced/sandbox-architecture.md:63-67` lists versions up to `1.6`
  and names only `conversation_open_channel`. The `message` frame example and
  the inbound frame table (line 95) do not list `agent_turn_id`.
- #310 (07e3dd0, 2026-10-23) added the outbound `secrets_hello` frame (`public_key`, `held`) and reads the `welcome` field `agent_id`, and kept `ProtocolVersion` at `1.5`. The frame tables in `docs/advanced/sandbox-architecture.md` (lines 94 and 104-114) list neither.
- `docs/release-notes/RELEASE-NOTES-1.51.0.md:31`: "The WebSocket protocol
  version stays `1.6`, and its major stays `1`."

## Impact

The version number no longer names the contract: a `1.6` client may or may
not read `agent_turn_id` or know the two delivery capabilities. The
architecture doc, which is where an implementer of the control host reads
the contract, records neither addition. Each later addition has to guess
which precedent to follow, so the drift will repeat.

## Recommendation

Make the spec state one rule. The remedy commit takes the explicit rule as
written: every addition, including a new welcome capability or a new
optional frame field, bumps the minor; a capability may gate the use of a
feature but does not replace the bump. It also adds a queued assertion for
the builder to bump to `1.7` and record the 1.51 additions in the
architecture doc. If the intent is the other reading (capability-gated
additions need no bump), replace the remedy commit with a spec edit that
says so and removes "Feature-flag negotiation per capability" from the
Rejected list.