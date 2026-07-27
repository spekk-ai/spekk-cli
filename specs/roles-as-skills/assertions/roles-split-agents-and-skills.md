---
id: roles-split-agents-and-skills
parent: roles-as-skills
created: 2026-07-27T00:00:00Z
priority: 1
status: done
branch: feat/roles-as-skills
---

# Install Writes Observer as an Agent and Coach/Builder as Skills

## Description

The install desired set changes. The observer stays an agent shim. The coach,
the builder, and the dev-loop become skills. The target descriptor must compute a
skill path for any skill name, not only for the dev-loop.

## Success Criteria

- The `target` descriptor computes a skill path for a given skill name, in both
  the global scope and the project scope. For a native-skill host (claude-code,
  opencode) the path is `<skills-dir>/<name>/SKILL.md`. For a command host
  (codex, cursor, copilot) the path is the command or prompt file for `<name>`. A
  host or scope that has no such location returns "".
- `desiredFiles` writes exactly one agent shim: the observer, in the agent
  directory, with the host's existing agent frontmatter.
- `desiredFiles` writes the coach, the builder, and the dev-loop as skills, at the
  skill paths for `spekk-coach`, `spekk-builder`, and `spekk-dev-loop`. A host or
  scope that returns "" for a skill path writes no file for that skill.
- `managedDirs` includes the agent directory and the directory of every skill
  path, so the scan finds both the observer shim and the skills.
- `go build ./...` passes. The `internal/install` tests are updated to this
  layout and pass.
