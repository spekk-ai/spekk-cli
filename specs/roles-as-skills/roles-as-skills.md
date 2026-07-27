---
id: roles-as-skills
created: 2026-07-27T00:00:00Z
priority: 2
---

# The Coach and Builder Roles Become Skills

## Overview

`spekk install` writes the coach, the builder, and the observer as agent shims
in the host tool's agent directory. An agent shim is a subagent. The dev loop
dispatches a subagent per role. A pilot (issue #154) showed that this dispatch is
the cost, because each subagent reads the codebase again from the start.

The new dev-loop skill drives one session that fetches each role with `spekk
prompt <role>`. It does not dispatch a subagent per role. So the coach and the
builder no longer need to be agents. This spec moves them to skills.

## Change

- The observer stays an agent shim. It is a separate, read-only, out-of-band
  concern, and that separation is the point.
- The coach and the builder become skills. The `spekk-dev-loop` skill stays a
  skill.
- The coach and builder skills stay thin: each one instructs the session to run
  `spekk prompt <role>` and to follow the output. This keeps the per-role
  overrides and lets a user run one role on its own.

## Migration

- The install reconciler (#155) prunes a stamped owned file that the desired set
  no longer contains. So a coach or builder agent shim that a reconciler wrote
  (stamped) is pruned when the desired set drops it.
- A user on an older version has an unstamped agent shim. The reconciler does not
  own an unstamped file, so it does not prune it. Therefore the install also
  recognizes the known legacy agent shim paths for the coach and the builder. If
  such a file is a spekk shim, the install makes a `.bak` backup and removes it.

## Non-goals

- This spec does not change the observer.
- This spec does not change the specs model or the prompt system.
