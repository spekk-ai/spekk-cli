---
id: layered-prompt-system
created: 2026-02-21T12:15:00Z
priority: 1
---

# Layered Prompt System

## Overview

Agent prompts are assembled from layers, allowing customization at global and project levels while preserving base behavior from the spekk package. Users can either extend the base prompt or completely override it.

## Naming Convention

Agent names are simplified: `coach`, `builder`, `observer` (not `coach-agent`, etc.).

- `<agent>.prompt.md` — **extends** the base prompt (appended after it)
- `<agent>.prompt.override.md` — **replaces** the base prompt entirely

## Layers

| Layer | Extend Path | Override Path | Purpose |
|-------|------------|---------------|---------|
| Base | `<spekk-package>/specs/<agent>-agent/<agent>.prompt.md` | — | Ships with spekk, core agent behavior |
| Global | `~/.config/spekk/<agent>.prompt.md` | `~/.config/spekk/<agent>.prompt.override.md` | User's personal defaults across all projects |
| Local | `.spekk/<agent>.prompt.md` | `.spekk/<agent>.prompt.override.md` | Project-specific customizations |

## Resolution

1. Determine base prompt:
   - If local override (`.spekk/<agent>.prompt.override.md`) exists → use as base
   - Else if global override (`~/.config/spekk/<agent>.prompt.override.md`) exists → use as base
   - Else use package base (`<spekk>/specs/<agent>-agent/<agent>.prompt.md`, required)
2. Append global extend (`~/.config/spekk/<agent>.prompt.md`) if exists
3. Append local extend (`.spekk/<agent>.prompt.md`) if exists
4. Concatenate with `\n\n---\n\n` separator

## Examples

### Extending: Adding company coding standards to builder

Create `~/.config/spekk/builder.prompt.md`:
```markdown
## Company Standards

- Use TypeScript strict mode
- All functions must have JSDoc comments
- No console.log in production code
```

This is appended to the base builder prompt for every project you work on.

### Extending: Adding project-specific context to coach

Create `.spekk/coach.prompt.md`:
```markdown
## Domain Knowledge

This is a healthcare app. When creating specs:
- Consider HIPAA compliance
- PHI must never be logged
- All dates must be in the account's timezone
```

This is appended to the base coach prompt for this project only.

### Overriding: Completely replacing the builder prompt

Create `.spekk/builder.prompt.override.md`:
```markdown
# Custom Builder Agent

You are a builder agent for a Django/React project.
...entirely custom instructions...
```

This replaces the base builder prompt entirely. Global and local extends still layer on top.

## Assertions

See `assertions/` for what must be true about layered prompt resolution.
