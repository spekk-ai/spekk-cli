---
id: cli-prompt-resolution
created: 2026-01-28T19:30:00Z
priority: 1
---

# CLI Agent Prompt Resolution

## Problem

When running `spekk coach` from a directory outside the spekk-cli installation (e.g., `~/myproject`), Claude Code cannot find the coach agent prompt file because it looks for `specs/coach-agent/coach-agent.prompt.md` relative to the current working directory.

The CLI should work from any directory while:
1. Running Claude Code in the user's working directory (so it can access their project files)
2. Making agent prompt files available to Claude Code from spekk-cli installation directory (NO copying to user directory)
3. Keeping the user's working directory clean and unmodified
4. Following the same pattern for all agent commands (coach, builder, observer)

## Requirements

- `spekk coach`, `spekk builder`, and `spekk observer` commands work from any directory
- `spekk` (spec parser) and `spekk --all` commands work from any directory  
- Claude Code runs in user's working directory to access their project files  
- Agent prompt files are read directly from spekk-cli installation directory (NO copying)
- Spec parser reads specs from spekk-cli installation directory, not current directory
- User's working directory remains clean and unmodified
- Solution works consistently across different environments and installation methods

## Success Criteria

- User can run `spekk coach` from `~/myproject` and Claude Code successfully loads the coach prompt from spekk-cli installation
- User can run `spekk` from `~/myproject` and get next assertion from spekk-cli specs
- Claude Code's working directory remains `~/myproject` so it can work with user's files
- No temporary files or directories created in user's working directory
- All commands (coach, builder, observer, parser) work consistently from any directory
- User's project directory structure is never modified by the CLI