---
id: global-prompt-optional
parent: layered-prompt-system
created: 2026-02-21T12:15:00Z
priority: 1
status: not_started
---

# Global Prompt Is Optional

## Description

The global prompt from `~/.spekk/` is optional. If present, it's appended to the base prompt. If missing, resolution continues silently.

## Success Criteria

- Global prompt loaded from `~/.spekk/specs/<agent>/<agent>.prompt.md`
- If file exists, content is appended to base prompt
- If file doesn't exist, no error - resolution continues
- `~` correctly expands to user's home directory on all platforms
