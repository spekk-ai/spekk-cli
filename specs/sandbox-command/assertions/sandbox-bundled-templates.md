---
id: sandbox-bundled-templates
parent: sandbox-command
created: 2026-03-12T18:00:00Z
priority: 1
status: not_started
depends-on: sandbox-command-routing
---

# Sandbox Bundled Templates

## Requirement

The cloud-init template and agent-client.py are bundled with the CLI package so the sandbox commands can provision and deploy without fetching from external repos.

## Success Criteria

- `src/sandbox/templates/` directory exists containing `cloud-init.yaml` and `agent-client.py`
- `cloud-init.yaml` matches the provisioning template from the spekk-agent-sandboxes repo (Docker, Node.js, git, gh, Claude Code CLI, systemd unit, agent user, `/opt/spekk/.provisioned` marker)
- `agent-client.py` matches the WebSocket agent client from the spekk-agent-sandboxes repo
- `src/sandbox/templates.js` exports a `getTemplatePath(filename)` function that resolves the absolute path to a bundled template file
- `src/sandbox/templates.js` exports a `readTemplate(filename)` function that reads and returns a template file's contents as a string
- The `cloud-init.yaml` template supports variable substitution for the SSH public key (replaces the placeholder `ssh-ed25519 AAAA... your-key-here` with the user's actual key)
- `package.json` `files` array includes `src/sandbox/templates/` so templates are included in the published npm package
