---
id: sandbox-cli-wiring
parent: sandbox-command
created: 2026-03-12T20:30:00Z
priority: 1
status: not_started
depends-on: sandbox-command-routing
branch: feature/sandbox-command
---

# Sandbox CLI Subcommand Wiring

## Requirement

All sandbox subcommands must be wired up in `src/sandbox/cli.js`. Currently only `create` and `deploy` are connected — `list`, `status`, `ssh`, and `destroy` print "not yet implemented" stubs.

## Success Criteria

- `src/sandbox/cli.js` `list` case imports and calls `listCommand` from `./list.js`
- `src/sandbox/cli.js` `status` case imports and calls `statusCommand` from `./status.js`, passing the sandbox name from `subArgs[0]`
- `src/sandbox/cli.js` `ssh` case imports and calls `sshCommand` from `./ssh.js`, passing `subArgs`
- `src/sandbox/cli.js` `destroy` case imports and calls `destroyCommand` from `./destroy.js`, passing `subArgs`
- No "not yet implemented" stubs remain in `cli.js`
- All six subcommands (create, list, status, ssh, destroy, deploy) execute their respective module when invoked
