---
id: verify-prompt-file-accessible
parent: cli-prompt-resolution
created: 2026-01-28T19:45:00Z
priority: 1
status: not_started
---

# Verify Prompt File Accessible

## Assertion

A test exists that verifies Claude Code can find and access the coach prompt file after prompt resolution.

## Success Criteria

- Test runs `spekk coach` from external directory
- Test verifies `specs/coach-agent/coach-agent.prompt.md` file is present in the working directory during Claude Code execution
- Test confirms Claude Code receives the agent activation message successfully
- Test verifies the copied prompt file contains the expected coach agent instructions
- Test confirms prompt files are cleaned up after Claude Code exits

## Context

This proves the core functionality - that the prompt resolution system actually makes the agent prompt files available to Claude Code when running from any directory.