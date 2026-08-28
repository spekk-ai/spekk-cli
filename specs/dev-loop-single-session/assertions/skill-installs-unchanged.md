---
id: skill-installs-unchanged
parent: dev-loop-single-session
created: 2026-07-27T00:00:00Z
priority: 2
status: done
depends-on: skill-single-session-flow
---

# The Rewritten Skill Still Embeds and Installs

## Description

The rewrite changes only the skill content. The file path, the frontmatter
shape, and the install path do not change. The skill must still embed in the
binary and install into every host tool.

## Success Criteria

- The skill file stays at the same path
  (`specs/install-spekk-dev-loop-skill/spekk-dev-loop-skill.md`), so the
  `//go:embed` reference and the install code find it with no change.
- The frontmatter keeps a `name: spekk-dev-loop` field and a `description` field,
  both well-formed YAML.
- `go build ./...` and the `internal/install` tests pass with the rewritten
  content.
- A manual `spekk install --target claude-code` (into a temporary home) writes
  the skill file, and the file holds the new single-session content.
