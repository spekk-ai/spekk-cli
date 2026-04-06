---
id: go-sandbox-commands
parent: golang-sandbox
created: 2026-04-05T12:29:00Z
priority: 2
status: done
depends-on: go-sandbox-api-client
branch: feature/golang-sandbox
---

# Go sandbox subcommands

All sandbox subcommands ported to Go with identical CLI interface.

## Success Criteria

**`spekk sandbox create --name <name>`:**
- Creates droplet with cloud-init that installs Claude Code agent
- Generates SSH key pair, uploads public key to DO
- Stores metadata locally in `~/.spekk/sandboxes/{name}.json`
- Supports `--region`, `--size`, `--project`, `--vpc` flags
- Auto-generates API token for agent communication
- Waits for droplet to be active, displays IP

**`spekk sandbox list`:**
- Lists all sandboxes from local metadata store with status

**`spekk sandbox status <name>`:**
- Shows detailed status of a sandbox (droplet info + agent status)

**`spekk sandbox ssh <name>`:**
- Opens SSH session to sandbox using stored key

**`spekk sandbox destroy <name>`:**
- Destroys droplet, removes SSH key from DO, removes local metadata

**`spekk sandbox deploy <name>`:**
- Downloads latest Go agent binary from GitHub releases
- SCPs binary to sandbox and restarts agent service

**All commands:**
- `--help` flag displays subcommand-specific help
- Proper error messages for missing args, missing sandbox, API failures
