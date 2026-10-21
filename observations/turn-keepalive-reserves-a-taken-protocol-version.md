---
slug: turn-keepalive-reserves-a-taken-protocol-version
type: outdated_specs
severity: medium
status: resolved
created: 2026-10-21T00:00:00Z
announced: 2026-10-21T19:02:38Z
affected:
  - specs/turn-progress-reporting/assertions/turn-keepalive-frame-sent-during-tool-calls.md
  - specs/turn-progress-reporting/assertions/sandbox-docs-record-turn-progress-protocol.md
  - specs/sandbox-sign-in-terminal/assertions/agent-sign-in-terminal.md
  - specs/turn-progress-reporting/assertions/sandbox-protocol-version-1-1.md
  - specs/turn-progress-reporting/turn-progress-reporting.md
  - cmd/sandbox/protocol.go
  - cmd/sandbox/protocol_test.go
  - docs/advanced/sandbox-architecture.md
---

# The Keepalive Assertion Reserved a Protocol Version That Is Already in Use

## Issue Description

The queued `turn-keepalive-frame-sent-during-tool-calls` assertion reserved protocol `1.3` and required a test that pinned that value. Protocol `1.3` already introduced the sign-in terminal. Protocol `1.4` added conversation channel routing, and `1.5` added the optional repository object. The code declares `1.5`. A builder that followed the old assertion could reduce the declared version and give one version two meanings.

The parent spec and the completed `sandbox-protocol-version-1-1`, `sandbox-docs-record-turn-progress-protocol`, and `agent-sign-in-terminal` assertions also contained fixed current-version requirements or the old keepalive reservation.

## Evidence

- `cmd/sandbox/protocol.go` declares `ProtocolVersion = "1.5"` and states the additive minor-version rule.
- `cmd/sandbox/protocol_test.go` checks the declared connection header and version compatibility. Commit df08e89 removed `TestProtocolVersionPinned`.
- `docs/advanced/sandbox-architecture.md` records the `1.1` through `1.5` additions and reserves `turn_keepalive` for a future change.
- The original queued assertion required `1.3` for the keepalive sender.
- The original completed assertions required current versions `1.1` or `1.3`, although the declared version is `1.5`.

## Impact

A reused protocol version cannot identify which frame types and fields a client supports. Fixed current-version requirements also make completed assertions false after a valid additive update.

## Resolution

The keepalive assertion now requires the next unused minor version at delivery. The parent spec uses the same rule. Completed assertions record the versions that introduced their features, and the documentation assertion requires the current declared version. The protocol tests check the actual declaration and compatibility.

The runtime remains at `1.5`. The keepalive sender and its in-flight tracking dependency remain `not_started`. The control host must accept and attribute the frame before a client sends it. This correction changes only the specs and this observation.
