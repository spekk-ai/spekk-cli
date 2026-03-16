---
id: base-prompt-required
parent: layered-prompt-system
created: 2026-02-21T12:15:00Z
priority: 1
status: not_started
---

# Base Prompt Is Required (Unless Overridden)

## Description

The base prompt from the spekk package is required unless a global or local override replaces it. If no base and no override exist, the agent fails to launch with a clear error.

## Success Criteria

- Base prompt loaded from `<spekk-package>/specs/<agent>-agent/<agent>.prompt.md`
- Internal prompt files are renamed from `<agent>-agent.prompt.md` to `<agent>.prompt.md` (e.g., `coach.prompt.md`)
- If base prompt is missing AND no override exists, error is thrown
- If base prompt is missing BUT an override exists, resolution succeeds
- Error message clearly identifies the missing file
- `PromptResolver` agent names are simplified: `coach`, `builder`, `observer`
