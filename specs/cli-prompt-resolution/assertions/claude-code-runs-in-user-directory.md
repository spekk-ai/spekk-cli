---
id: claude-code-runs-in-user-directory
parent: cli-prompt-resolution
created: 2026-01-28T19:30:00Z
priority: 1
status: not_started
---

# Claude Code Runs in User Directory

## Assertion

When launching agent commands via spekk CLI, Claude Code runs in the user's current working directory, not in the spekk-cli installation directory.

## Success Criteria

- Running `spekk coach` from `~/thinknimble/vuenome` sets Claude Code's working directory to `~/thinknimble/vuenome`
- Claude Code can access and modify files in the user's project directory
- User can work with their project files through Claude Code without changing directories
- The CLAUDE.md file from the user's project is loaded (if it exists) rather than the spekk-cli CLAUDE.md

## Context

This ensures users can work on their own projects while still having access to the spekk agent prompts. The CLI should not force users to work from within the spekk-cli directory.