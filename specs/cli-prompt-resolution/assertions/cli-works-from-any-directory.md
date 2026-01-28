---
id: cli-works-from-any-directory
parent: cli-prompt-resolution
created: 2026-01-28T19:30:00Z
priority: 1
status: not_started
---

# CLI Works From Any Directory

## Assertion

All spekk CLI commands (`spekk coach`, `spekk builder`, `spekk observer`) work when run from any directory, not just from within the spekk-cli installation directory.

## Success Criteria

- Running `spekk coach` from `~/thinknimble/vuenome` successfully launches Claude Code with coach prompt
- Running `spekk builder` from any external project directory works correctly
- Running `spekk observer` from any external project directory works correctly
- No "file not found" errors related to agent prompt files when running from external directories

## Context

Currently, when running from an external directory, Claude Code cannot locate the agent prompt files because they are resolved relative to the current working directory instead of the spekk-cli installation directory.