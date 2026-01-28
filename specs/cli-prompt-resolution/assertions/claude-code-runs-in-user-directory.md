---
id: claude-code-runs-in-user-directory
parent: cli-prompt-resolution
created: 2026-01-28T19:30:00Z
priority: 1
status: in_progress
---

# Claude Code Runs in User Directory

## Assertion

When launching agent commands via spekk CLI, Claude Code runs in the user's current working directory, not in the spekk-cli installation directory.

## Success Criteria

- Test verifies Claude Code's working directory matches where `spekk coach` was run
- Test confirms Claude Code can access files in the user's project directory  
- Test verifies user's project CLAUDE.md is loaded (not spekk-cli's CLAUDE.md)
- Integration tests prove Claude Code doesn't run from spekk-cli installation directory

**Tests:** src/cli/__tests__/working-directory-verification.test.js

**Tests Required:**
- `verify-claude-working-directory` assertion must be completed  
- Test should verify file access and working directory programmatically

## Context

This ensures users can work on their own projects while still having access to the spekk agent prompts. The CLI should not force users to work from within the spekk-cli directory.