---
id: observation-output-contract
parent: observer-skill-discovery
created: 2026-05-22T12:00:00Z
priority: 2
status: not_started
---

# Observation Output Contract Is Documented And Followed

## Description

All observer modes (default loop and skills) write observation files following a shared contract: per-mode subdirectories, required frontmatter fields, required body sections. The contract is convention-enforced (documented in the observer prompt and seed skill) — not parser-enforced. A separate spec will promote it to parser-enforcement later, alongside analogous validation for coach and builder outputs.

## Success Criteria

- `specs/observer-agent/observer.prompt.md` documents the observation output contract: per-mode subdirectories, required frontmatter, required body sections
- Default loop writes to `observations/default/` (not the flat `observations/` root) — the observer prompt reflects this change
- Each skill specifies its own subdirectory under `observations/{skill-name}/`
- Required frontmatter fields documented: `id`, `created`, `skill`, `type`, `severity`, `affected_specs`, `affected_files`
- Required body sections documented: Issue Description / Evidence / Impact / Recommendation
- The contract states that observation `type` values are extensible — skills may introduce new types
- The seed `coverage-gap` skill follows the contract as a working example
- Observer prompt forbids writing outside `observations/` (preserves read-only contract — no code, no specs)
- A note in the spec references the future validation work that will enforce this across all three agents
