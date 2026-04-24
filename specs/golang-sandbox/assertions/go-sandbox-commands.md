---
id: go-sandbox-commands
parent: golang-sandbox
created: 2026-04-05T12:29:00Z
priority: 2
status: done
depends-on: go-sandbox-api-client
branch: feature/golang-migration
---

# Go sandbox subcommands

All sandbox subcommands ported to Go with identical CLI interface.

## Success Criteria

**`spekk sandbox create --name <name>`:**
- A new DigitalOcean droplet is provisioned and accessible via SSH
- A dedicated SSH key pair exists for the sandbox and is registered with DigitalOcean
- Agent credentials (API token, AWS, GitHub) are injected and the agent service is ready
- Sandbox metadata is persisted locally for use by other subcommands
- `--region`, `--size`, `--project`, `--vpc` flags are supported

**`spekk sandbox list`:**
- All known sandboxes are displayed with name, IP, region, status, and creation date

**`spekk sandbox status <name>`:**
- Droplet status, provisioning state, and agent service health are displayed

**`spekk sandbox ssh <name>`:**
- An interactive SSH session opens to the sandbox using its dedicated key

**`spekk sandbox destroy <name>`:**
- The droplet, its SSH key on DigitalOcean, the local key files, and local metadata are all removed
- Confirmation is required unless `--force` is passed

**`spekk sandbox deploy <name>`:**
- The latest release binary is installed on the sandbox and the agent service is restarted

**All commands:**
- `--help` flag displays subcommand-specific help
- Clear error messages for missing arguments, unknown sandboxes, and API failures
