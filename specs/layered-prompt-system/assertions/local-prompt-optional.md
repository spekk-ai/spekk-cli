---
id: local-prompt-optional
parent: layered-prompt-system
created: 2026-02-21T12:15:00Z
priority: 1
status: not_started
---

# Local Prompt Is Optional

## Description

The local prompt from the current working directory is optional. If present, it's appended after global. If missing, resolution continues silently.

## Success Criteria

- Local prompt loaded from `./specs/<agent>/<agent>.prompt.md` (relative to cwd)
- If file exists, content is appended after global prompt
- If file doesn't exist, no error - resolution continues
- Works correctly regardless of where spekk is invoked from
