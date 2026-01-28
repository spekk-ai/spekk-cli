---
id: agent-prompts-accessible-anywhere
parent: cli-prompt-resolution
created: 2026-01-28T19:30:00Z
priority: 1
status: in_progress
---

# Agent Prompts Accessible Anywhere

## Assertion

Agent prompt files from the spekk-cli installation are accessible to Claude Code regardless of the user's current working directory.

## Success Criteria

- Coach agent prompt (`specs/coach-agent/coach-agent.prompt.md`) is found by Claude Code when run from any directory
- Builder agent prompt (`specs/builder-agent/builder-agent.prompt.md`) is found by Claude Code when run from any directory  
- Observer agent prompt (`specs/observer-agent/observer-agent.prompt.md`) is found by Claude Code when run from any directory
- Claude Code receives the correct "You are the [Agent] Agent - read the prompt and follow the instructions exactly" message and can access the corresponding prompt file

**Tests Required:**
- `verify-prompt-file-accessible` assertion must be completed
- Unit tests in `src/cli/__tests__/prompt-resolution.test.js` (✅ completed)
- Integration tests proving end-to-end prompt access

## Constraints

**❌ PROHIBITED APPROACH:**
- Copying prompt files to user's working directory (messy, pollutes user's project)
- Creating temporary files or directories in user's project space
- Any approach that modifies the user's working directory structure

**✅ REQUIRED APPROACH:**
- Read prompt files directly from spekk-cli installation directory
- Keep user's working directory clean and unmodified
- Claude Code runs in user's directory but accesses prompts from installation

## Context

The CLI needs to make these prompt files available to Claude Code through one of these approaches:
- Pass prompt content directly via stdin (along with agent activation message)
- Use Claude Code's memory/instruction system to load prompt content
- Set environment variables with prompt file paths for Claude Code to read
- Send prompt file path in agent activation message for Claude Code to read directly