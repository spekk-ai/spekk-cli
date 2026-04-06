---
id: go-prompt-resolver
parent: golang-cli
created: 2026-04-05T12:12:00Z
priority: 1
status: done
depends-on: go-command-router
branch: feature/golang-cli
---

# Go prompt resolver implements layered prompt resolution

The Go prompt resolver loads agent prompts using the same layered resolution as the Node implementation.

## Success Criteria

**Base prompt resolution (first match wins):**
- Local override: `.spekk/{agent}.prompt.override.md`
- Global override: `~/.spekk/{agent}.prompt.override.md`
- Package base: `specs/{agent}-agent/{agent}.prompt.md` (relative to spekk installation)

**Extension layers (appended in order):**
- Global extend: `~/.spekk/{agent}.prompt.md`
- Local extend: `.spekk/{agent}.prompt.md`

**Behavior:**
- Layers concatenated with `\n\n---\n\n` separator
- Missing override/extend files silently skipped
- Missing package base prompt is a fatal error
- `createActivationMessage()` wraps prompt with working directory and installation path context
- Works for all agents: coach, builder, observer
