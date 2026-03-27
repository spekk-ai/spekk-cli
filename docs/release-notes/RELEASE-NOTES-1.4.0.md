# Spekk CLI 1.4.0 — Layered Prompts, Sandboxes, WebSocket Server

This release covers changes since v1.2.4 (the last announced release).

## Layered Prompt System (v1.4.0, PR #21)

Agent prompts can now be customized at two levels without modifying the core spekk package:

- **Global** (`~/.spekk/`): Personal defaults that apply across all projects
- **Local** (`.spekk/`): Project-specific customizations

### Two modes

- `<agent>.prompt.md` — **extends** the base prompt (appended after it)
- `<agent>.prompt.override.md` — **replaces** the base prompt entirely

### Resolution order

1. Determine base: local override > global override > package base
2. Append global extend if exists
3. Append local extend if exists

### Example: Extending the builder with project rules

Create `.spekk/builder.prompt.md` in your project:

```markdown
## Project Rules

- Use TypeScript strict mode
- All functions must have JSDoc comments
```

This is appended to the base builder prompt for this project only. The `.spekk/` directory can be committed or gitignored per team preference.

### Simplified agent names

Agent names have been simplified from `coach-agent` / `builder-agent` / `observer-agent` to just `coach` / `builder` / `observer` across the codebase and prompt file names.

## Sandbox Command (v1.3.0, PR #46)

New `spekk sandbox` command for managing cloud sandbox environments:

- `spekk sandbox create` / `list` / `status` / `ssh` / `destroy` / `deploy`
- Automatic API token generation for sandbox environments
- Token auto-injection during sandbox creation

## WebSocket Server (v1.2.7, PR #39)

New `spekk serve` command that starts a WebSocket server for the browser extension, enabling real-time communication between the CLI and the Spekk web UI.

## Other changes

- Builder locking tests (#34)
- Fixed missing coach-skills-system in package files (#38)
- Fixed interactive mode (#43)
- Self-hosted GitHub Actions runner for CI (#44)
- Fixed publish workflow (curl missing in node:20-slim container)

## Upgrade

```bash
npm install @spekk/cli@1.4.0
```
