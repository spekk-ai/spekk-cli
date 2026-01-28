---
id: verify-claude-working-directory
parent: cli-prompt-resolution
created: 2026-01-28T19:45:00Z
priority: 1
status: done
---

# Verify Claude Working Directory

## Assertion

A test exists that verifies Claude Code runs in the user's current working directory when launched via spekk CLI.

## Success Criteria

- Test creates a temporary directory with a test file (e.g., `test-project.txt`)
- Test runs `spekk coach` from that directory  
- Test verifies Claude Code can see and access the test file in its working directory
- Test confirms Claude Code working directory matches the directory where spekk was run
- Test verifies Claude Code does NOT run from the spekk-cli installation directory

**Tests:** src/cli/__tests__/working-directory-verification.test.js

## Context

This ensures users can work on their own project files through Claude Code while still having access to spekk agent prompts. Critical for the user experience.