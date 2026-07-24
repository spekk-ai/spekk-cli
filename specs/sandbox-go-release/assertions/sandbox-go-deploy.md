---
id: sandbox-go-deploy
parent: sandbox-go-release
created: 2026-04-01T00:00:00Z
priority: 1
status: done
branch: feature/sandbox-go-release
depends-on: sandbox-release-downloader
---

# Sandbox Create and Deploy Share One Go Deploy Path

Both `sandbox create` and `sandbox deploy` deploy the Go agent binary through a
single shared function. No Python, uv, or venv steps remain.

## Success Criteria

- `internal/sandbox/commands.go` defines
  `deployAgent(ip, keyPath, name string, artifacts *releaseArtifacts)` which:
  - scp's `artifacts.BinaryPath` to `root@{ip}:/opt/spekk/agent-client`
  - runs one remote script over SSH that `chmod +x /opt/spekk/agent-client`,
    creates `/opt/spekk/workspace`, chowns `/opt/spekk` to `agent:agent`, and
    creates `/var/log/spekk` owned by `agent:agent`
  - writes the Go systemd unit to `/etc/systemd/system/spekk-agent.service`
    (ExecStart `/opt/spekk/agent-client`, stdout/stderr appended to
    `/var/log/spekk/agent.log`)
  - runs `systemctl daemon-reload && systemctl enable spekk-agent && systemctl restart spekk-agent`
- `Create` (in `commands.go`) fetches artifacts via
  `fetchReleaseArtifacts("latest")`, renders the embedded cloud-init with the
  sandbox's public key (`renderCloudInit`) and passes it to `createDroplet` as
  droplet user-data, then calls `deployAgent(ip, keyPath, name, artifacts)` with
  the already-fetched artifacts
- `Deploy` (in `commands.go`) fetches artifacts via
  `fetchReleaseArtifacts("latest")` and calls the same `deployAgent(...)`
- No calls to `uv`, `pip`, or `websockets`, and no `src/sandbox/**` JS files,
  remain in the sandbox source
