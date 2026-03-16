---
id: local-prompt-optional
parent: layered-prompt-system
created: 2026-02-21T12:15:00Z
priority: 1
status: in_progress
locked-by: builder-Williams-MBP.local-88202-1773689679
---

# Local Prompt Is Optional

## Description

The local prompt from the project's `.spekk/` directory is optional. If present, it's appended after global. If missing, resolution continues silently.

## Success Criteria

- Local extend prompt loaded from `.spekk/<agent>.prompt.md` (relative to cwd)
- Local override prompt loaded from `.spekk/<agent>.prompt.override.md` (relative to cwd)
- If override exists, it replaces the base prompt (takes precedence over global override)
- If extend exists, content is appended after global extend
- If neither exists, no error — resolution continues
- Works correctly regardless of where spekk is invoked from
