---
id: dev-loop-single-session
created: 2026-07-27T00:00:00Z
priority: 2
---

# Dev Loop — One Session Plays the Roles in Turn

## Overview

The `spekk-dev-loop` skill describes four subagent stages, each a fresh
dispatch: coach, coordinate, builders, review. A pilot compared this multi-agent
loop with a single session that builds a whole feature. On features that fit in
one session, the single session used about five times fewer tokens, cost about
three times less, and finished about three times faster, with the same result
quality. (See issue #154.)

This spec rewrites the skill. The new skill drives one continuous session that
plays the roles in turn. The session scopes, builds, and verifies in the same
context. It does not dispatch a separate subagent for each role.

## Mechanism

The session fetches each role's instructions with `spekk prompt <role>` at the
start of that phase. This reuses the existing prompt system, so the role
overrides still apply and a user can still run one role on its own. The single
session, not a new mechanism, is the change.

## Non-goals

- This spec does not change where the role files live. The coach and builder
  agent shims stay for now. The move from agents to skills is separate work, and
  the install reconciler (#155) will do that migration.
- This spec does not remove the ability to run one role on its own.
