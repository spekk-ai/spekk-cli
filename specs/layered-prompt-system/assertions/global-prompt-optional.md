---
id: global-prompt-optional
parent: layered-prompt-system
created: 2026-02-21T12:15:00Z
priority: 1
status: done
---

# Global Prompt Is Optional

**Tests:** src/cli/__tests__/prompt-resolver.test.js

## Description

The global prompt from `~/.spekk/` is optional. If present, it's appended to the base prompt. If missing, resolution continues silently.

## Success Criteria

- Global extend prompt loaded from `~/.spekk/<agent>.prompt.md`
- Global override prompt loaded from `~/.spekk/<agent>.prompt.override.md`
- If override exists, it replaces the package base prompt
- If extend exists, content is appended after the base (or overridden base)
- If neither exists, no error — resolution continues
- `~` correctly expands to user's home directory on all platforms
