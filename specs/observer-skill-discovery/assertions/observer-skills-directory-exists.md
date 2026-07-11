---
id: observer-skills-directory-exists
parent: observer-skill-discovery
created: 2026-05-22T12:00:00Z
priority: 2
status: done
branch: feature/extend-observer
---

# Observer Skills Package Directory Exists With Coverage-Gap Seed

## Description

`specs/observer-skills/` exists as the package-shipped skill directory for observer, containing the `coverage-gap` seed skill. This skill provides a different lens from the default observer loop — instead of checking spec→code (what specs declare vs. what's implemented), it checks code→spec (what code exists with no spec backing it).

**Tests:** embedded_test.go

## Success Criteria

- Directory `specs/observer-skills/` exists in the repository
- Directory contains `coverage-gap-skill.md` with valid frontmatter (`id: coverage-gap`, `description`)
- Coverage-gap skill follows the structure used by coach skills: Triggers / Workflow / Validation / Examples sections
- Skill workflow describes scanning `internal/` for code regions (packages, exported types, exported functions) that no assertion in `specs/` references
- Skill outputs a consolidated observation to `observations/coverage-gap/YYYY-MM-DDTHH-MM-SSZ.md` following the observation output contract
- Seed skill is discoverable via `spekk observer --help` and invocable via `spekk observer coverage-gap`
- The directory is included in the embedded FS at build time so the skill ships with the binary
