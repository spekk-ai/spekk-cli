---
id: skill-resolver-tested
parent: layered-skill-discovery
created: 2026-03-28T12:00:00Z
priority: 1
status: done
depends-on: skill-resolver-layered-resolution
---

# SkillResolver Has Comprehensive Test Coverage

## Description

The `SkillResolver` is tested using isolated temp directories for all three layers. Tests cover resolution order, override semantics, aliases, frontmatter id matching, listing, and deduplication. Coach and builder CLI tests are updated to reflect the new interfaces.

## Success Criteria

- `src/cli/__tests__/skill-resolver.test.js` exists with tests for:
  - Resolution from package, global, and local directories
  - Local overrides global, local overrides package
  - Legacy alias resolution (`meeting` → `meeting-notes-to-specs-skill`)
  - Frontmatter `id` field resolution
  - `listSkills()` returns skills from all layers, deduplicated
  - Non-`.md` files are ignored
  - `null` returned for unknown skills
  - Constructor defaults for `homeDir` and `cwd`
- `src/coach/__tests__/cli.test.js` does not import `SKILL_MAP` or `resolveSkillContent`
- `src/builder/__tests__/cli.test.js` includes tests for `extractSkillArg` and `buildBuilderSkillMessage`
- All tests use temp directories for isolation (no side effects on real file system)
