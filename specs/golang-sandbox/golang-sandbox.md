---
id: golang-sandbox
created: 2026-04-05T12:28:00Z
priority: 2
---

# Go Sandbox Command

Port the `spekk sandbox` subcommands from Node.js to Go.

Manages DigitalOcean cloud sandbox environments. Already has a Go-based deploy agent that runs on the droplet — this spec ports the CLI-side orchestration.

## Current Architecture

- `src/sandbox/cli.js` — subcommand routing (125 lines)
- `src/sandbox/create.js` — droplet creation with cloud-init (282 lines)
- `src/sandbox/do-api.js` — DigitalOcean API client (91 lines)
- `src/sandbox/store.js` — local metadata store in `~/.config/spekk/sandboxes/` (44 lines)
- `src/sandbox/tokens.js` — token management (10 lines)
- `src/sandbox/list.js`, `status.js`, `ssh.js`, `destroy.js`, `deploy.js`, `release.js` — subcommand implementations
- `src/sandbox/agent.js`, `templates.js` — agent/template management

## Strategy

- Port DigitalOcean API client to Go (net/http)
- Port metadata store to Go (JSON files in `~/.config/spekk/sandboxes/`)
- Port each subcommand as a Go function
- Cloud-init template embedded via Go embed
