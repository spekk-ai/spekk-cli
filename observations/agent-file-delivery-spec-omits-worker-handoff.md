---
slug: agent-file-delivery-spec-omits-worker-handoff
type: outdated_specs
severity: medium
status: open
created: 2026-11-27T12:00:00Z
announced: 2026-11-27T20:03:41Z
affected:
  - specs/agent-file-delivery/agent-file-delivery.md
  - cmd/sandbox/invoke.go
  - cmd/sandbox/conn_holder.go
  - cmd/spekk/conversation.go
  - specs/agent-file-delivery/assertions/agent-current-thread-reply.md
  - specs/agent-file-delivery/assertions/conversation-files-command.md
---

# Agent file delivery spec omits current-thread reply and the worker-to-CLI handoff

## Issue Description
The `agent-file-delivery` parent spec says the CLI lets a sandbox agent "publish an artifact or open a Slack conversation with files". Since 1.51 it can also reply in the current Slack thread (`spekk conversation reply`), and the parent spec does not mention that capability.

The assertions under the spec also describe behavior that depends on a contract no spec states. The CLI cannot read the welcome frame or the `message` frame. The sandbox worker reads both and passes what it found to the Claude child as environment variables. The CLI then decides from those variables. No spec names these variables or says when the worker sets them.

## Evidence
- `cmd/sandbox/invoke.go:96-104` sets `SPEKK_AGENT_SESSION_ID` from `msg.AgentSessionID`, `SPEKK_AGENT_TURN_ID` from `msg.AgentTurnID`, and `SPEKK_CONVERSATION_DELIVERY` / `SPEKK_CURRENT_THREAD_DELIVERY` to `1` or `0` from the capabilities that the welcome frame advertised (`cmd/sandbox/conn_holder.go:93-118`). The worker sets them once, when the turn starts.
- `cmd/spekk/conversation.go:177-183` and `:245-258` make every delivery decision (direct versus spool, capability refusal, no-thread refusal) from those variables alone.
- `agent-current-thread-reply.md` says "If the welcome frame lacks the current thread capability, the CLI returns a clear error" and "If the current turn has no turn ID". `conversation-files-command.md` says "The app advertises direct conversation delivery in its welcome frame". Neither assertion says how the CLI learns these facts.
- `grep -rl 'SPEKK_AGENT_TURN_ID\|SPEKK_AGENT_SESSION_ID\|SPEKK_CURRENT_THREAD_DELIVERY' specs` matches nothing. `SPEKK_CONVERSATION_DELIVERY` and the two capability names appear only in the release-note and protocol-version assertions, which do not state the handoff.

## Impact
Two binaries share this handoff, and no assertion covers the worker side. A change to `invoke.go` that drops or renames one variable passes every assertion. It also breaks `conversation open` delivery or `conversation reply` without notice: for example, a missing `SPEKK_AGENT_TURN_ID` makes every reply say that the turn has no Slack thread. A reader of the parent spec also does not learn that current-thread reply exists.

## Recommendation
Update the parent spec to name current-thread reply as a capability. State the handoff there too: which variables the worker sets for each turn, what each one means, and that the worker sets them once when the turn starts. The remedy commit makes this spec edit. A person can then decide whether the worker side needs its own assertion.