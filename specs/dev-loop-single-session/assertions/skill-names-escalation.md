---
id: skill-names-escalation
parent: dev-loop-single-session
created: 2026-07-27T00:00:00Z
priority: 2
status: done
depends-on: skill-single-session-flow
---

# The Skill Names When to Escalate to Sub-agents

## Description

The single session is the default. It is not the only path. A feature can be too
large for one coherent session. The skill states when to escalate, so the reader
does not fragment small work but still has the tool for large work.

## Success Criteria

- The skill states that the single session is the default for a feature that fits
  in one session.
- The skill names the escalation trigger: a feature too large for one coherent
  session (context strain or loss of coherence), or a workflow the user asks for.
- For the escalation case only, the skill keeps the parallel-builders pattern:
  builders in separate worktrees, and the orchestrator integrates and verifies
  their work.
- The skill frames escalation as the exception, not the default.
