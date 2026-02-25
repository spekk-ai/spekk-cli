---
id: base-prompt-required
parent: layered-prompt-system
created: 2026-02-21T12:15:00Z
priority: 1
status: not_started
---

# Base Prompt Is Required

## Description

The base prompt from the spekk package is always required. If it's missing, the agent fails to launch with a clear error.

## Success Criteria

- Base prompt loaded from `<spekk-package>/specs/<agent>/<agent>.prompt.md`
- Error thrown if base prompt file doesn't exist
- Error message clearly identifies the missing file
