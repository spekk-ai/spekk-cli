---
id: integration-test-external-directory
parent: cli-prompt-resolution
created: 2026-01-28T19:45:00Z
priority: 1
status: done
---

# Integration Test External Directory

## Assertion

An integration test exists that verifies `spekk coach` works correctly when run from an external directory.

## Success Criteria

- Test creates a temporary external directory (e.g., `/tmp/test-spekk-external`)
- Test runs `spekk coach` command from that directory using spawn/execSync
- Test captures that the command exits successfully (exit code 0)
- Test verifies no "file not found" errors related to prompt files
- Test verifies Claude Code initialization message appears
- Test exists in the test suite and passes

## Context

This integration test proves that the prompt resolution actually works in the real scenario that was failing before - running spekk coach from outside the installation directory.