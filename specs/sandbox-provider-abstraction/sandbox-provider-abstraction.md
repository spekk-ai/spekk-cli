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

`SandboxMeta` gains `Provider string`, which names the provider that owns the machine. The change is additive: an entry written before the field existed reads as DigitalOcean, so an existing fleet keeps working and keeps being destroyable.

A provider's own identifiers stay in named fields — `dropletId`, `sshKeyId` — rather than in one opaque encoded handle. An opaque handle was tried first and rejected: it forced a breaking rename of fields that live in every operator's `sandboxes.json`, and its empty value could not distinguish "no machine to destroy" from "the identifier did not survive the load". That ambiguity is the difference between a clean teardown and a droplet that bills forever. Concrete fields cost one leaked name per provider; the second cloud provider adds its own, and the generic layer still reads none of them. Provider-specific teardown happens inside the provider's Destroy, not in generic sandbox code.

## Provider interface shape (informational)

The interface needs at minimum:
- `Name` — the provider name recorded in metadata
- `Create` — provision a VM and record its address and identifiers
- `Destroy` — tear down the VM
- `Status` — fetch live VM status from the provider

SSH key management is provider-internal. DO uploads keys to its API; manual expects the user's key is already authorized.

## CLI surface

`spekk sandbox create --provider <name>` selects the provider. Provider-specific flags (region, size, VPC, project) are scoped to their provider — passing `--region` to the manual provider is an error.

`spekk sandbox destroy` and `status` read the provider from stored metadata and dispatch accordingly. `deploy` and `ssh` need only an IP and a key, so they stay provider-blind.

## Open question: is "manual" a provider?

A manual machine has no lifecycle to own. Modeling it as a `Provider` gives it a `Create` that registers, a `Destroy` that cannot destroy, and a `Status` with no API to ask — three methods that describe an absence. It may fit better as a provisioning mode on `create`, with `provider: none` in metadata, leaving `Provider` to mean "the cloud that owns this machine". A manual sandbox is a full member of the registry either way; the question is only which type models it. This is unresolved, and the manual assertions stay `not_started` until it is.
