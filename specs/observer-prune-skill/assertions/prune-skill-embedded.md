---
id: prune-skill-embedded
parent: observer-prune-skill
created: 2026-07-25T12:00:00Z
priority: 1
status: done
depends-on: prune-skill-markdown-exists
branch: feature/observer-prune-skill
---

# Prune Skill Is Embedded In The Binary

## Description

The `//go:embed` directive in `embedded.go` is an **explicit-path** embed (not
a glob), so a new skill file is invisible to the shipped binary until its path
is added. Without this, `spekk observer prune` resolves only from a source
checkout and fails for installed users.

**Tests:** embedded_test.go — mirror the existing
`TestEmbeddedFS_ObserverCoverageGapSkill`, reading
`specs/observer-skills/prune-skill.md` from the embedded FS and asserting it is
present with `id: prune`.

## Success Criteria

- The `//go:embed` line at `embedded.go:11` includes the path
  `specs/observer-skills/prune-skill.md` alongside the existing embedded skill
  paths.
- The embedded filesystem serves that path at runtime: reading
  `specs/observer-skills/prune-skill.md` via the embedded FS succeeds and its
  content contains `id: prune`.
- No other embedded paths are removed from the directive.
