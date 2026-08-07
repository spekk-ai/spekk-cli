---
id: works-for-all-agents
parent: layered-prompt-system
created: 2026-02-21T12:15:00Z
priority: 1
status: done
depends-on: prompts-concatenated-with-separator
---

# Works For All Agents

## Description

Layered prompt resolution works for all agent types via a shared `PromptResolver`.

## Success Criteria

- Coach agent supports layered prompts
- Builder agent supports layered prompts
- Observer agent supports layered prompts
- Any future agents automatically support layered prompts via `PromptResolver`
- All agents use simplified names (`coach`, `builder`, `observer`) in the resolver
