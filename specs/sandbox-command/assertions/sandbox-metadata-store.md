---
id: sandbox-metadata-store
parent: sandbox-command
created: 2026-03-12T15:00:00Z
priority: 1
status: done
branch: feature/sandbox-command
depends-on: sandbox-command-routing
---

# Sandbox Metadata Store

## Requirement

Sandbox metadata is persisted locally in `~/.spekk/sandboxes.json` so the CLI can map sandbox names to droplet IDs, IPs, and other state without querying DO every time.

## Success Criteria

- `src/sandbox/store.js` exports functions: `loadSandboxes`, `saveSandbox`, `removeSandbox`, `getSandbox`
- `~/.spekk/` directory is created automatically if it does not exist
- `sandboxes.json` stores an object keyed by sandbox name with values: `{ dropletId, ip, region, size, createdAt, status }`
- `saveSandbox(name, data)` merges data into the existing entry (or creates a new one)
- `removeSandbox(name)` deletes the entry from the JSON file
- `getSandbox(name)` returns the entry or `null` if not found
- `loadSandboxes()` returns the full object (empty `{}` if file does not exist)
- File reads and writes use `fs.promises` with proper error handling for missing files
