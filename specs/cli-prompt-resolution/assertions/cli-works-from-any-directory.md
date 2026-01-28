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

- Integration tests verify `spekk coach`, `spekk builder`, and `spekk observer` work from external directories
- Commands exit successfully (code 0) when run from any directory
- No "file not found" errors related to agent prompt files
- All agent types work consistently across different working directories

**Tests Required:**
- `integration-test-external-directory` assertion must be completed
- Similar tests for builder and observer agents

## Context

Currently, when running from an external directory, Claude Code cannot locate the agent prompt files because they are resolved relative to the current working directory instead of the spekk-cli installation directory.