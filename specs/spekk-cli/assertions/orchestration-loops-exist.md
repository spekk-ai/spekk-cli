---
id: orchestration-loops-exist
parent: spekk-cli
created: 2026-01-21T19:15:00Z
priority: 1
status: done
---

# Orchestration Loops Exist

## Requirement

`spekk coach` and `spekk builder` commands provide full orchestration workflows that automate the spec-driven development cycle.

## Success Criteria

### Builder Loop (`spekk builder`):
✅ Gets next priority assertion via `spekk next`
✅ Launches builder agent with assertion context
✅ Waits for agent completion 
✅ Automatically commits changes with proper commit message
✅ Repeats cycle until no more assertions
✅ Handles interrupts gracefully (Ctrl+C)
✅ Provides colored output and logging

### Coach Loop (`spekk coach`):  
✅ Launches coach agent in interactive mode
✅ Waits for user input and spec creation
✅ Automatically commits new specs
✅ Returns to coach for next interaction
✅ Continues until user exits
✅ Handles interrupts gracefully

### Workflow Integration:
✅ Builder loop integrates with Claude Code agent sessions
✅ Coach loop integrates with Claude Code agent sessions  
✅ Proper error handling when agents fail
✅ Git integration for automatic commits
✅ Status logging and progress indicators

## Historical Context

Original builder-loop.sh workflow:
- Parse next task from JSON
- Extract task ID, title, file path
- Invoke builder agent on specific assertion
- Git commit with formatted message
- Loop until complete

This full orchestration is essential for automated spec-driven development.

**Tests:** src/cli/__tests__/orchestration-loops.test.js