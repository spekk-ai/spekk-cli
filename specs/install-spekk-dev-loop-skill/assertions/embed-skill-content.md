---
id: embed-skill-content
parent: install-spekk-dev-loop-skill
created: 2026-07-15T00:00:00Z
priority: 1
status: not_started
---

# The `spekk-dev-loop` Skill Content Ships in the Binary

The skill's `SKILL.md` content is embedded in the binary so it can be written
to disk on install, following the same `//go:embed` pattern already used for
the coach/observer skills.

## Success Criteria

- A verbatim copy of the skill exists at
  `specs/install-spekk-dev-loop-skill/spekk-dev-loop-skill.md` — frontmatter
  `name: spekk-dev-loop` (Claude Code `SKILL.md` convention uses `name`, not
  `id`) plus the full body. This file is the single source of truth written to
  disk on install.
- `embedded.go`'s `//go:embed` directive includes
  `specs/install-spekk-dev-loop-skill/spekk-dev-loop-skill.md`, and the project
  still builds (`go build ./cmd/spekk`).
- `spekk.EmbeddedFS` can read that path and returns the exact bytes of the
  source file — no transformation, trimming, or re-rendering.

## Tests

Add a case to `embedded_test.go` mirroring `TestEmbeddedFS_ObserverCoverageGapSkill`:
`fs.ReadFile(EmbeddedFS, "specs/install-spekk-dev-loop-skill/spekk-dev-loop-skill.md")`
succeeds and the content contains the `name: spekk-dev-loop` frontmatter line.

**Note:** This assertion only makes the content ship in the binary. Reading it
from `internal/install` and writing it to disk during install is
`install-writes-skill-for-claude-code`.
