---
id: cli-prompt-resolution
created: 2026-01-28T19:30:00Z
priority: 1
status: not_started
---

# CLI Agent Prompt Resolution

## Problem

When running `spekk coach` from a directory outside the spekk-cli installation (e.g., `~/thinknimble/vuenome`), Claude Code cannot find the coach agent prompt file because it looks for `specs/coach-agent/coach-agent.prompt.md` relative to the current working directory.

The CLI should work from any directory while:
1. Running Claude Code in the user's working directory (so it can access their project files)
2. Making agent prompt files available to Claude Code regardless of working directory
3. Following the same pattern for all agent commands (coach, builder, observer)

## Requirements

- `spekk coach`, `spekk builder`, and `spekk observer` commands work from any directory
- Claude Code runs in user's working directory to access their project files  
- Agent prompt files are accessible to Claude Code from spekk-cli installation directory
- Solution works consistently across different environments and installation methods

## Success Criteria

- User can run `spekk coach` from `~/thinknimble/vuenome` and Claude Code successfully loads the coach prompt
- Claude Code's working directory remains `~/thinknimble/vuenome` so it can work with user's files
- All agent commands (coach, builder, observer) work consistently from any directory
- No hardcoded paths or environment-specific assumptions