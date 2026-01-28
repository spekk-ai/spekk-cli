---
id: verify-prompt-file-accessible
parent: cli-prompt-resolution
created: 2026-01-28T19:45:00Z
priority: 1
status: done
---

# Verify Prompt File Accessible

**Tests:** src/cli/__tests__/verify-prompt-file-accessible.test.js

## Assertion

A test exists that verifies Claude Code can find and access the coach prompt file after prompt resolution.

## Success Criteria

- Test runs `spekk coach` from external directory
- Test verifies NO `specs/` directory is created in the user's working directory
- Test confirms Claude Code receives the agent activation message successfully  
- Test verifies Claude Code can access coach prompt from spekk-cli installation directory
- Test confirms user's working directory remains completely unmodified
- Test verifies no temporary files or cleanup needed in user's directory

## Context

This proves the core functionality - that the prompt resolution system actually makes the agent prompt files available to Claude Code when running from any directory.