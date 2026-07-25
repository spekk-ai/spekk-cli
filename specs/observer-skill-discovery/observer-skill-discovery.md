---
id: observer-skill-discovery
created: 2026-05-22T12:00:00Z
priority: 2
---

# Observer Skill Discovery

## Overview

The observer agent supports layered skill discovery — exactly like coach and builder do today via `SkillResolver`. Users can drop observer-specific skills at the local (`.spekk/skills/observer/`) or global (`~/.config/spekk/skills/observer/`) layer, and package-shipped skills in `specs/observer-skills/` serve as the fallback.

Layered prompts already work for observer (via `PromptResolver`). This spec closes the remaining gap: skill discovery, invocation, and a structured output contract for observation files.

## Background

Previously, observer had no skill support at all:
- `internal/cli/skill.go` `packageSkillDirNames` only mapped `coach` and `builder`
- `internal/agent/observer.go` `RunObserver` never checked for a skill subcommand
- `spekk observer --help` used a hardcoded help string instead of dynamic skill listing
- Observations were a flat directory with no per-mode separation

This made the observer the only agent that couldn't be extended with project-specific or user-specific skills.

## Architecture

### Resolution (mirrors coach/builder)

| Layer | Path | Purpose |
|-------|------|---------|
| Local | `.spekk/skills/observer/*.md` | Project-specific observer skills |
| Global | `~/.config/spekk/skills/observer/*.md` | User's personal observer skills |
| Package | `specs/observer-skills/*.md` | Ships with spekk |

First match wins. `SkillResolver` already implements this layering — observer just needs to be registered in `packageSkillDirNames`.

### Invocation

`spekk observer [skill] [flags]` — if the first positional arg resolves to an observer skill, the skill content is inlined into the activation message via `BuildSkillMessage`. Otherwise the observer runs in its default mode (timer-based polling loop, configurable via `--interval`).

### Help

`spekk observer --help` dynamically lists available observer skills by calling `SkillResolver.ListSkills("observer")`, replacing the hardcoded help string. This matches the coach/builder pattern via the shared `ShowHelp` helper in `internal/agent/launcher.go`.

### Observation Output Contract

All observer modes (default loop and skills) write observation files. The contract is **convention-enforced, not parser-enforced** for now — a separate validation spec will promote this to parser-enforcement once the format stabilizes across all three agents.

**Directory structure:**
- Default loop writes to `observations/default/YYYY-MM-DDTHH-MM-SSZ.md`
- Each skill writes to `observations/{skill-name}/YYYY-MM-DDTHH-MM-SSZ.md`
- Per-mode subdirectories keep observations navigable and scoped

**Required frontmatter:**
```yaml
---
id: {kebab-case-id}              # unique within the skill subdirectory
created: 2026-05-22T15:30:00Z    # ISO 8601, UTC
skill: {skill-name | "default"}  # which mode produced this
type: {drift-type}               # see allowed values
severity: low | medium | high
affected_specs: []               # list of spec IDs, can be empty
affected_files: []               # list of file paths, can be empty
---
```

**Allowed `type` values (extensible per skill):**
- `code_spec_misalignment` — default loop
- `outdated_specs` — default loop
- `compression_opportunity` — default loop
- `spec_conflicts` — default loop
- `coverage_gap` — coverage-gap skill (code a spec could optionally document)
- `prune_candidate` — prune skill (deletion / consolidation candidates)
- Future skills register their own types in their skill markdown

**Required body sections:**
- `## Issue Description`
- `## Evidence` (concrete file paths, line numbers, or excerpts — no vague claims)
- `## Impact`
- `## Recommendation`

**Output rules for skills:**
- If a skill writes anything, it writes to `observations/{skill-name}/` using the format above
- Skills MAY produce one consolidated observation per scan or multiple — skill's choice
- Skills that don't write files (read-only summaries, interactive Q&A) are valid and don't need a subdirectory
- Skills MUST NOT write outside `observations/` (the observer's read-only contract still holds — no code, no specs)

### Package Skills

`specs/observer-skills/` ships with at least one seed skill so the resolver has real content to find, and users have a working example. Initial seed: `coverage-gap` (finds code in `internal/` that no assertion references — the inverse lens from the default loop's spec→code checks).

## Future Work

Once the observation contract stabilizes through use, a separate spec (`observation-validation` or similar) will:
- Promote the frontmatter and body conventions to parser-enforced rules
- Add `spekk` CLI commands to validate, query, and clean observations
- Apply analogous validation to coach skill outputs (`projects/`, new specs) and builder skill outputs (code changes)
- Cover validation rules for all three agents uniformly

## Assertions

See `assertions/` for what must be true about observer skill discovery and the observation output contract.
