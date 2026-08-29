---
id: provider-interface
parent: sandbox-provider-abstraction
created: 2026-08-29T20:00:00Z
priority: 1
status: not_started
branch: dev/headless-sandbox
---

# A Provider interface abstracts VM lifecycle from sandbox orchestration

## Success Criteria

- A `Provider` interface in `internal/sandbox/` defines `Create`, `Destroy`, and `Status` methods
- `Create` accepts provider-specific config and returns the VM's public IP (or error)
- `Destroy` accepts an instance ID and tears down all provider-managed resources (VM, SSH keys, etc.)
- `Status` accepts an instance ID and returns live VM state from the provider
- `SandboxMeta` stores `Provider string` and `InstanceID string` instead of `DropletID int` and `SSHKeyID int`
- `SandboxMeta.InstanceID` is opaque to the generic layer — only the provider interprets it
- Provider-specific teardown state (e.g., DO SSH key IDs) is the provider's responsibility to track, not stored in generic metadata
- Sandbox lifecycle functions (`Create`, `Destroy`, `Status` in `commands.go`) accept a `Provider` rather than constructing a DO client inline
