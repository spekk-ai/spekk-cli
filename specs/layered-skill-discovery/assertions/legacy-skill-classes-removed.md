---
id: legacy-skill-classes-removed
parent: layered-skill-discovery
created: 2026-03-28T12:00:00Z
priority: 2
status: done
---

# Legacy Skill OOP Classes Are Removed

## Description

The unused `Skill` base class (`src/coach/skill-interface.js`) and its re-export (`src/coach/skills/index.js`) are deleted. The hardcoded `skillsDir` reference in `PromptResolver.createActivationMessage()` is also removed — skills are now inlined by the CLI, the agent doesn't need to discover them.

## Success Criteria

- `src/coach/skill-interface.js` does not exist
- `src/coach/skills/index.js` does not exist
- `src/coach/skills/` directory does not exist
- `src/coach/__tests__/skill-interface.test.js` does not exist
- `PromptResolver.createActivationMessage()` does not include a `Skills directory:` line in its output
- `specs/builder-skills/` directory exists as an empty placeholder (with `.gitkeep`)
- `package.json` `files` array includes `specs/builder-skills/`
