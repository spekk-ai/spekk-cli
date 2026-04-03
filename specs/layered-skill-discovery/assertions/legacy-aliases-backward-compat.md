---
id: legacy-aliases-backward-compat
parent: layered-skill-discovery
created: 2026-03-28T12:00:00Z
priority: 1
status: done
---

# Legacy Aliases Preserve Backward Compatibility

## Description

Existing CLI subcommands (`spekk coach meeting`, `spekk coach coordinate`) continue to work via legacy alias mappings in the `SkillResolver`. The old `SKILL_MAP` and `resolveSkillContent()` function are removed from `src/coach/cli.js`.

## Success Criteria

- `spekk coach meeting` resolves to `meeting-notes-to-specs-skill` via legacy alias
- `spekk coach coordinate` resolves to `coordinator-skill` via legacy alias
- No `SKILL_MAP` constant exists in `src/coach/cli.js`
- No `resolveSkillContent()` function exists in `src/coach/cli.js`
- Meeting-specific transcript handling works (coach-specific behavior preserved)
- `src/serve/index.js` uses `SkillResolver` for coordinator skill resolution
