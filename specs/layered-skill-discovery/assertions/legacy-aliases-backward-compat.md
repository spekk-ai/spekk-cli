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
- `SKILL_MAP` constant no longer exists in `src/coach/cli.js`
- `resolveSkillContent()` function no longer exists in `src/coach/cli.js`
- Meeting-specific transcript handling still works (coach-specific behavior preserved)
- The `src/serve/index.js` coordinator skill reference uses `SkillResolver` instead of the removed `resolveSkillContent`
