---
id: layered-skill-discovery
created: 2026-03-28T12:00:00Z
priority: 1
---

# Layered Skill Discovery System

## Overview

Skills for coach and builder agents are discovered from layered directories — exactly like agent prompts work today via `PromptResolver`. Users can add their own skills at the local (project) or global (home) level, and package-shipped skills serve as the fallback.

Previously, skills were hardcoded in a `SKILL_MAP` in `src/coach/cli.js` and the builder had no skill support at all. This spec replaces that with a `SkillResolver` class that mirrors the `PromptResolver` pattern.

## Architecture

### SkillResolver

A new `SkillResolver` class in `src/cli/skill-resolver.js` provides:
- `resolveSkill(agentName, subcommand)` — returns `{ name, content, source }` or `null`
- `listSkills(agentName)` — returns `[{ name, source }]`

Constructor accepts `{ homeDir, cwd }` for testability (same as `PromptResolver`).

### Resolution Order (first match wins)

| Layer | Path | Purpose |
|-------|------|---------|
| Local | `.spekk/skills/{agent}/*.md` | Project-specific skills |
| Global | `~/.config/spekk/skills/{agent}/*.md` | User's personal skills across all projects |
| Package | `specs/coach-skills-system/` (coach) / `specs/builder-skills/` (builder) | Ships with spekk |

### Skill Name Derivation

- Primary: filename stem minus `.md` (e.g., `my-validator.md` → subcommand `my-validator`)
- Secondary: frontmatter `id` field as alternative lookup name

### Legacy Aliases

For backward compatibility with existing CLI subcommands:
- `meeting` → `meeting-notes-to-specs-skill`
- `coordinate` → `coordinator-skill`

### Skill Activation

When a skill resolves, its full markdown content is inlined into the activation message within `<skill-content>` tags. The agent receives the complete workflow instructions directly — same pattern as the existing coach skill inlining.

## All Agents Support Skills

- **Coach:** `spekk coach <skill>` (replaces `SKILL_MAP` approach)
- **Builder:** `spekk builder <skill> [flags]` (first positional arg is checked as skill)
- **Observer:** `spekk observer <skill> [flags]` (added via `observer-skill-discovery` spec; package directory is `specs/observer-skills/`)

Adding a future agent type requires only registering it in `validAgents` (prompts), `packageSkillDirNames` (skills), and `legacyAliases` — no other code paths special-case agent names.

## Dynamic Help

`spekk coach --help`, `spekk builder --help`, and `spekk observer --help` all dynamically list available skills by calling `skillResolver.listSkills()` via the shared `ShowHelp` helper in `internal/agent/launcher.go`.

## Assertions

See `assertions/` for what must be true about layered skill discovery.
