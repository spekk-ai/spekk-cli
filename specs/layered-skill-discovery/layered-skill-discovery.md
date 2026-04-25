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
| Global | `~/.spekk/skills/{agent}/*.md` | User's personal skills across all projects |
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

## Both Agents Support Skills

- **Coach:** `spekk coach <skill>` (replaces `SKILL_MAP` approach)
- **Builder:** `spekk builder <skill> [flags]` (new — first positional arg is checked as skill)

## Dynamic Help

Both `spekk coach --help` and `spekk builder --help` dynamically list available skills by calling `skillResolver.listSkills()`.

## Assertions

See `assertions/` for what must be true about layered skill discovery.
