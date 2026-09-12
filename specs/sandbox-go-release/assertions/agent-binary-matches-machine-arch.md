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
- `fetchAgentBinary` runs `detectArch` then `downloadAgentBinary(arch)`, and is
  invoked after the machine is reachable in all three paths: Create and
  Provision (via `equipSandbox`) and `Deploy`.
- A detection failure surfaces as a clear error; the empty `BinaryPath` is
  cleaned up regardless (the `defer` removes whatever path is set, including
  none).

**Note:** the `uname -m` → GOARCH mapping and the download split currently have
no unit test in `internal/sandbox`. The package's tests and `go build ./...`
pass, and the arm64 build cross-compiles, but the arch mapping itself is
unverified by a test — a regression that mis-maps an architecture would ship
silently. A test covering the mapping (accepted values and the unsupported-arch
error) is worth adding.
