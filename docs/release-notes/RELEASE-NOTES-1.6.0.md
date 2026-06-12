# Spekk CLI 1.6.0 — Use Spekk from Any Coding Assistant

This release covers changes since v1.5.0.

## Install Spekk Agents into Your Coding Assistant (PR #111)

New `spekk install` command registers the spekk agents (coach, builder, observer) as subagents in your existing coding assistant:

```bash
spekk install --target claude-code   # or: cursor, copilot, opencode, codex
```

The installed agents are thin shims that fetch their instructions from the spekk binary at session start, so they always match your installed spekk version. Ask **spekk-coach** to draft specs and **spekk-builder** to implement them, right inside your normal session.

Supporting commands:

- `spekk prompt <agent>` — print an agent's fully resolved prompt (honors `.spekk/` overrides and extensions)
- `spekk skill list <agent>` / `spekk skill show <agent> <name>` — expose layered skill discovery on the command line, so agents in any harness can fetch skill content on demand

## Other Changes

- CI moved from self-hosted runners to `ubuntu-latest` for public-repo safety

## Upgrade

```bash
spekk update
```

Or download the latest binary from [GitHub Releases](https://github.com/spekk-ai/spekk-cli/releases).
