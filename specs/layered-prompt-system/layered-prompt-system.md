---
id: layered-prompt-system
created: 2026-02-21T12:15:00Z
priority: 1
---

# Layered Prompt System

## Overview

Agent prompts are assembled from three layers, allowing customization at global and project levels while preserving base behavior from the spekk package.

## Layers

| Layer | Path | Purpose |
|-------|------|---------|
| Base | `<spekk-package>/specs/<agent>/<agent>.prompt.md` | Ships with spekk, core agent behavior |
| Global | `~/.spekk/specs/<agent>/<agent>.prompt.md` | User's personal defaults across all projects |
| Local | `./specs/<agent>/<agent>.prompt.md` | Project-specific customizations |

## Resolution

1. Start with base prompt (required, ships with spekk)
2. Append global prompt if exists
3. Append local prompt if exists
4. Concatenate with `\n\n---\n\n` separator

## Examples

### Adding company coding standards to builder

Create `~/.spekk/specs/builder-agent/builder-agent.prompt.md`:
```markdown
## Company Standards

- Use TypeScript strict mode
- All functions must have JSDoc comments
- No console.log in production code
```

This applies to every project you work on.

### Adding project-specific skills to coach

Create `./specs/coach-agent/coach-agent.prompt.md`:
```markdown
## Domain Knowledge

This is a healthcare app. When creating specs:
- Consider HIPAA compliance
- PHI must never be logged
- All dates must be in patient's timezone
```

This only applies to this project.

## Assertions

See `assertions/` for what must be true about layered prompt resolution.
