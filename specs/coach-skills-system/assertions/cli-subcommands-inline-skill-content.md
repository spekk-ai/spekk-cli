---
id: cli-subcommands-inline-skill-content
parent: coach-skills-system
created: 2026-02-25T20:15:00Z
priority: 1
status: not_started
---

# CLI Subcommands Inline Skill Content from Markdown Files

When a CLI subcommand activates a skill (e.g., `spekk coach meeting`, `spekk coach coordinate`), the CLI reads the corresponding skill markdown file from `specs/coach-skills-system/` and inlines its full content into the activation message. The coach agent receives the complete workflow instructions directly — it does not need to discover or read the skill file itself.

## What Must Be True

### Generic Skill Resolution
- A mapping exists from subcommand name to skill filename (e.g., `meeting` → `meeting-notes-to-specs-skill.md`, `coordinate` → `coordinator-skill.md`)
- The skill file is resolved relative to the spekk installation's `specs/coach-skills-system/` directory (same path resolution as `prompt-resolver.js`)
- If the skill file does not exist at the resolved path, a clear error is shown and the process exits (consistent with existing missing-file error patterns in cli.js)

### Activation Message Contains Skill Content
- The activation message includes the full markdown content of the skill file, clearly delimited (e.g., within a labeled section)
- The activation instruction tells the coach to follow the inlined workflow immediately — not to go read a file
- Existing behavior (transcript file handling for `meeting`, etc.) continues to work after the skill content

### No Hardcoded Workflow Steps in JS
- Workflow steps are NOT hardcoded as JS string concatenation in `src/coach/cli.js`
- The JS code only handles: subcommand detection, skill file resolution/reading, and subcommand-specific args (like transcript files)
- All workflow content comes from the skill markdown files — single source of truth

### Extensible Pattern
- Adding a new CLI subcommand for a skill requires only: adding the subcommand-to-filename mapping and any subcommand-specific arg handling
- No workflow duplication between JS and markdown

## Success Criteria

- `spekk coach meeting` activation message contains the full content of `meeting-notes-to-specs-skill.md`
- `spekk coach coordinate` activation message contains the full content of `coordinator-skill.md` (when PR #25 merges)
- Coach agent reliably activates the meeting skill workflow when launched via `spekk coach meeting`
- Skill markdown files remain the single source of truth for workflow instructions
- Tests validate that skill content is present in the activation message
