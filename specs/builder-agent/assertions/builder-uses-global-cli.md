---
id: builder-uses-global-cli
parent: builder-agent
created: 2026-01-28T21:10:00Z
priority: 1
status: done
---

# Builder Uses Global CLI Commands

## What Must Be True

The builder agent prompt must instruct builders to use the global `spekk` CLI commands instead of local npm scripts when working on external projects.

## Required Updates

The builder prompt at `specs/builder-agent/builder-agent.prompt.md` must:

1. Replace all instances of `npm run next` with `spekk next`
2. Use global CLI commands consistently throughout the workflow
3. Explain that `spekk` is the global CLI tool for spec-driven development

## Specific Changes

**In "Get Next Task" section (around line 22):**
- ❌ OLD: `npm run next`
- ✅ NEW: `spekk next`

**In "Validate System Health" section (around line 117):**
- ❌ OLD: `npm run next`  
- ✅ NEW: `spekk next`

**In bootstrap instructions:**
- Should mention using `spekk next` when the parser doesn't exist yet

## Success Criteria

- ✅ All references to `npm run next` replaced with `spekk next`
- ✅ Builder prompt explains that spekk is the global CLI tool
- ✅ Instructions are consistent throughout the prompt
- ✅ No local npm script references remain for spec parsing

## Impact

This change ensures builders working on external projects use the standardized global CLI instead of project-specific npm scripts. The spekk CLI is designed to work from any directory with a specs/ folder.