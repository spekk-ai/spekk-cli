---
id: skill-single-session-flow
parent: dev-loop-single-session
created: 2026-07-27T00:00:00Z
priority: 1
status: done
branch: feat/dev-loop-single-session
---

# The Skill Describes One Session with Role Phases

## Description

The rewritten `spekk-dev-loop` skill tells the reader to run one continuous
session. The session plays the roles in turn. It does not dispatch a separate
subagent for each role by default.

## Success Criteria

- The skill body describes one session, not four subagent stages. It names three
  phases in order: scope (coach), build, and verify.
- Each phase step says to run `spekk prompt <role>` and to follow the output in
  the same session (for example `spekk prompt coach` for the scope phase and
  `spekk prompt builder` for the build phase).
- The build phase implements the assertions in dependency order within the same
  session, and reuses the context from the scope phase rather than re-derive the
  code.
- The skill drops the separate `coordinate` stage. One session that scopes its
  own work does not need a second independent pass.
- The `description` field in the frontmatter states the single-session flow, so
  the host tool delegates for the right task.
