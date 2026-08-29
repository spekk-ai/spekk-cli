---
id: sandbox-provider-abstraction
created: 2026-08-29T20:00:00Z
priority: 1
---

# Sandbox Provider Abstraction

Sandboxes currently hard-code DigitalOcean as the only infrastructure provider. The provisioning layer (cloud-init, SSH, agent deploy, credential injection) is already cloud-agnostic — it only needs an IP address and an SSH key. The DigitalOcean coupling lives entirely in "get me a VM" and "tear it down."

This spec introduces a provider interface so sandboxes work with any infrastructure: DigitalOcean, AWS, bare metal, a Raspberry Pi — anything reachable over SSH. A "manual" provider is the simplest and most universal path: the user supplies an IP and SSH key, spekk provisions and deploys.

## Architecture

Two concerns, cleanly separated:

- **Provider** — creates and destroys the VM (cloud API, or no-op for manual). Returns an IP. Provider-specific config (region, size, VPC) lives inside the provider, not in the generic sandbox layer.
- **Sandbox** — provisions the OS, deploys the agent, injects credentials, manages SSH sessions. Takes an IP. Unchanged from today.

`SandboxMeta` becomes provider-agnostic: `Provider string` identifies which provider manages the VM, `InstanceID string` is an opaque provider-scoped identifier (DO droplet ID as string, empty for manual). Provider-specific teardown (e.g., deleting DO SSH keys) happens inside the provider's Destroy, not in generic sandbox code.

## Provider interface shape (informational)

The interface needs at minimum:
- `Create` — provision a VM, return its public IP
- `Destroy` — tear down the VM (no-op for manual)
- `Status` — fetch live VM status from the provider (SSH-only for manual)

SSH key management is provider-internal. DO uploads keys to its API; manual expects the user's key is already authorized.

## CLI surface

`spekk sandbox create --provider <name>` selects the provider. Provider-specific flags (region, size, VPC, project) are scoped to their provider — passing `--region` to the manual provider is an error.

`spekk sandbox destroy`, `status`, `deploy`, `ssh` read the provider from stored metadata and dispatch accordingly.
