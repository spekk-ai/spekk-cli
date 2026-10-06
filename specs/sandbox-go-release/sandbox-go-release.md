---
id: sandbox-go-release
created: 2026-04-01T00:00:00Z
priority: 1
---

# Sandbox Go Release Integration

The sandbox commands deploy the Go agent binary by downloading it as a versioned
release asset (`sandbox-linux-{arch}`) from this project's own public GitHub
repository (`spekk-ai/spekk-cli`). `.github/workflows/publish.yml` builds and
publishes both `sandbox-linux-amd64` and `sandbox-linux-arm64` on each `v*` /
`exp-*` tag, and spekk deploys the build matching the sandbox machine's CPU
(detected over SSH), not the operator's. The cloud-init template ships embedded
in the CLI binary (`//go:embed cloud-init.yaml` in `internal/sandbox/embed.go`)
rather than as a separately downloaded asset, so no control-host infrastructure
is duplicated into or fetched from a private repo.

The release the artifacts come from is the latest published release by default;
`--release <tag>` on create/provision/deploy pins a specific one (e.g. an
`exp-*` prerelease that carries an architecture the latest release does not).
