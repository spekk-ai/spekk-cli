---
id: prompts-concatenated-with-separator
parent: layered-prompt-system
created: 2026-02-21T12:15:00Z
priority: 1
status: not_started
depends_on:
  - base-prompt-required
  - global-prompt-optional
  - local-prompt-optional
---

# Prompts Concatenated With Separator

## Description

When multiple prompt layers exist, they are concatenated with a clear separator so the agent can distinguish between layers.

## Success Criteria

- Layers concatenated in order: base (or override) → global extend → local extend
- Separator between layers is `\n\n---\n\n`
- Final prompt is a single string passed to Claude
- No separator added if only one layer exists
