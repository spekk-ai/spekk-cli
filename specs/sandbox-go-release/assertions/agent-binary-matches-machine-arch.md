---
id: agent-binary-matches-machine-arch
parent: sandbox-go-release
created: 2026-09-12T00:00:00Z
priority: 1
status: done
branch: sandbox-agent-arm64
---

# The Deployed Agent Binary Matches the Sandbox Machine's CPU

The agent runs on the sandbox machine, so the binary's architecture is the
machine's, not the operator's. An arm64 host (a Raspberry Pi, an arm64 droplet)
is served an arm64 build; an amd64 host is served amd64. spekk never deploys a
binary the machine cannot run and then reports success.

## Success Criteria

- `.github/workflows/publish.yml` builds `./cmd/sandbox` for both
  `GOARCH=amd64` and `GOARCH=arm64` and publishes `sandbox-linux-amd64` and
  `sandbox-linux-arm64` as release assets.
- `detectArch` runs `uname -m` on the sandbox over SSH and maps it to a GOARCH:
  `x86_64`/`amd64` → `amd64`, `aarch64`/`arm64` → `arm64`.
- Any other `uname -m` value returns an error naming the unsupported
  architecture, rather than falling back to a default build.
- `detectArch` reads the machine name from stdout only. ssh writes its own warnings to stderr, for example on a host whose sshd has no post-quantum key exchange, and a combined read refuses a supported machine by the text of the warning.
- `fetchAgentBinary` runs `detectArch` and then calls `selectArch(arch)`, which picks the builds for that architecture and downloads nothing. Create and Provision call it through `equipSandbox` after the machine is reachable. Deploy also calls it after the machine is reachable. The machine then downloads the builds for that architecture itself (`specs/sandbox-spekk-binaries/`, `machine-downloads-the-release`).
- A detection failure returns an error. Create, Provision, and Deploy call `cleanup` when they finish. `cleanup` removes the temp directory when one exists. `downloadBinaries`, which runs only when the machine cannot download the builds, removes its temp directory if a download fails.

**Tests:** `internal/sandbox/release_test.go` tests supported and unsupported architecture values in `TestArchFromUname`, SSH warnings in `TestDetectArchIgnoresSSHWarnings`, and the agent and CLI asset names in `TestDownloadBinariesSelectsArchAssets`.
