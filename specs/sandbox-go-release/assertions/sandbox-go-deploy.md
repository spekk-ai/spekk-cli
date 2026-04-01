---
id: sandbox-go-deploy
parent: sandbox-go-release
created: 2026-04-01T00:00:00Z
priority: 1
status: done
branch: feature/sandbox-go-release
depends-on: sandbox-release-downloader
---

# Sandbox Deploy Uses Go Binary

Both `sandbox create` and `sandbox deploy` use a single shared function to deploy the Go agent binary. No Python, uv, or venv steps remain.

## Success Criteria

- `src/sandbox/agent.js` exports `deployAgent(ip, artifacts?)` which:
  - Uses provided `artifacts` or calls `fetchReleaseArtifacts()` to get the binary path and version
  - rsyncs the binary to `root@{ip}:/opt/spekk/agent-client`
  - SSHes to set `chmod +x /opt/spekk/agent-client`
  - Creates `/var/log/spekk` with `agent:agent` ownership
  - Writes the Go systemd unit file to `/etc/systemd/system/spekk-agent.service` (replacing any legacy Python unit), with stdout/stderr logging to `/var/log/spekk/agent.log`
  - SSHes to run `systemctl daemon-reload && systemctl restart spekk-agent`
  - Prints the deployed version (e.g. `Agent v1.2.3 deployed`)
- `src/sandbox/create.js` calls `deployAgent(ip, releaseArtifacts)` instead of the inline `deployAgentClient` function, reusing artifacts already fetched for cloud-init
- `src/sandbox/deploy.js` calls `deployAgent(ip)` instead of its inline deploy logic
- `src/sandbox/create.js` passes `cloudInitPath` (from `fetchReleaseArtifacts()`) to `createDroplet` as user data instead of reading the bundled template
- No calls to `uv`, `pip`, or `websockets` remain in any sandbox source file
- `src/sandbox/templates/cloud-init.yaml` does not exist
