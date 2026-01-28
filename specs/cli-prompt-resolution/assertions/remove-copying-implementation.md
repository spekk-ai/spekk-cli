---
id: remove-copying-implementation
parent: cli-prompt-resolution
created: 2026-01-28T20:00:00Z
priority: 1
status: not_started
---

# Remove Copying Implementation

## Assertion

The current `PromptResolver` implementation that copies prompt files to user's working directory is removed and replaced with a clean approach.

## Success Criteria

- `src/cli/prompt-resolver.js` is deleted or completely rewritten to not copy files
- `withPromptFiles()` function does not create any files or directories in user's working directory
- All agent CLI files (`src/coach/cli.js`, `src/builder/cli.js`, `src/observer/cli.js`) use new approach
- Unit tests in `src/cli/__tests__/prompt-resolution.test.js` are updated to test new approach
- No `copyFileSync`, `mkdirSync`, or file copying logic remains in the codebase

## Context

The current implementation copies prompt files to the user's working directory, which pollutes their project space. This violates the requirement to keep the user's directory clean and unmodified.

## Alternative Approaches

The new implementation should use one of these approaches instead:
- Pass prompt content via stdin along with agent activation message
- Set environment variable with prompt file path for Claude Code to read
- Use Claude Code's memory/instruction system
- Modify agent activation message to include prompt file path