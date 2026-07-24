---
id: sandbox-go-release
created: 2026-04-01T00:00:00Z
priority: 1
---

# Sandbox Go Release Integration

The sandbox commands deploy the Go agent binary by downloading it as a versioned
release asset (`sandbox-linux-amd64`) from this project's own public GitHub
repository (`spekk-ai/spekk-cli`). That asset is built and published by
`.github/workflows/publish.yml` on each `v*` / `exp-*` tag. The cloud-init
template ships embedded in the CLI binary (`//go:embed cloud-init.yaml` in
`internal/sandbox/embed.go`) rather than as a separately downloaded asset, so no
control-host infrastructure is duplicated into or fetched from a private repo.
