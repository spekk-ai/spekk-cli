---
id: agent-prompts-accessible-anywhere
parent: cli-prompt-resolution
created: 2026-01-28T19:30:00Z
priority: 1
status: not_started
---

# Agent Prompts Accessible Anywhere

## Assertion

Agent prompt files from the spekk-cli installation are accessible to Claude Code regardless of the user's current working directory.

## Success Criteria

- Coach agent prompt (`specs/coach-agent/coach-agent.prompt.md`) is found by Claude Code when run from any directory
- Builder agent prompt (`specs/builder-agent/builder-agent.prompt.md`) is found by Claude Code when run from any directory  
- Observer agent prompt (`specs/observer-agent/observer-agent.prompt.md`) is found by Claude Code when run from any directory
- Claude Code receives the correct "You are the [Agent] Agent - read the prompt and follow the instructions exactly" message and can access the corresponding prompt file

## Context

The CLI needs to make these prompt files available to Claude Code through one of these potential approaches:
- Copying prompt files to user directory temporarily
- Setting up symlinks to prompt files
- Using Claude Code's memory/instruction system
- Modifying how Claude Code resolves prompt file paths