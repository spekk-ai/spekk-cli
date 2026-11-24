---
slug: reply-assertion-omits-non-slack-turn-error
type: underspecified_spec
severity: medium
status: open
created: 2026-11-24T12:00:00Z
announced: 2026-11-24T20:02:46Z
affected:
  - specs/agent-file-delivery/assertions/agent-current-thread-reply.md
  - cmd/spekk/conversation.go
  - cmd/sandbox/invoke.go
---

# Reply assertion does not say what happens in a turn with no Slack thread

## Issue Description
The `agent-current-thread-reply` assertion covers a missing capability, but it does not say what `spekk conversation reply` does when the current turn has no turn ID, for example a sandbox turn that did not come from Slack. The implementation answers `Error: the current agent turn ID is not ready; try again`. A retry can never succeed: the worker sets the turn ID once, in the child environment, when the turn starts. An agent that follows the message retries a call that fails the same way every time.

## Evidence
- `cmd/sandbox/invoke.go:104` sets `SPEKK_AGENT_TURN_ID=`+`msg.AgentTurnID` when it starts the Claude child. `cmd/sandbox/message.go:9` declares `agent_turn_id` with `omitempty`, so a turn whose frame has no turn ID gets an empty value for the whole turn.
- `cmd/spekk/conversation.go:256-258` returns `the current agent turn ID is not ready; try again` when that variable is empty.
- `docs/cli-reference.md:1172` says the command "works only in a sandbox turn that came from Slack".
- `specs/agent-file-delivery/assertions/agent-current-thread-reply.md` covers the capability refusal and partial results, but not the case where the turn ID is absent.

Both of these meet the assertion: the current "not ready; try again" error, and an error that says the turn has no Slack thread to reply to. To an agent, the first means "retry" and the second means "use `conversation open` instead".

## Impact
An agent in a turn that did not come from Slack is told to retry an operation that cannot succeed. It can loop or stall instead of choosing `spekk conversation open`. The wording "not ready" also suggests a race, which sends a person who reads the transcript to look in the wrong place.

## Recommendation
State in the assertion that when the current turn has no turn ID, the CLI sends no request and fails with an error that says the turn has no current Slack thread and names `spekk conversation open` as the alternative, with no instruction to retry. The proposed remedy commit adds this sentence and sets the assertion to `failed`, so the builder adjusts the message.