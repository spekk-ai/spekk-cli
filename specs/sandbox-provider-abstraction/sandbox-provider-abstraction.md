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

- **Provider** — creates and destroys the machine and reports its state. Nil when no cloud owns it. The create-time settings (region, size, VPC, project) travel in one typed `CreateOptions` struct that every provider receives and reads selectively. A string map was tried and rejected: it put each provider in charge of its own defaults with no way to return them, so omitting `--region` left the metadata blank.
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

## A machine nobody owns is not a provider

"Manual" was modeled as a `Provider` first, and the type never fitted. Its `Create` only copied two flags into metadata, its `Destroy` could not destroy, and its `Status` had no API to ask, so it returned nothing. Three methods describing an absence. The generic layer gave it away: it string-matched `provider == "manual"` around every lifecycle event, which is the shape polymorphism is supposed to remove.

So `Provider` means "the cloud that owns this machine", and a machine no cloud owns records `provider: none` and has no `Provider` at all. `Create` takes a nil provider and registers instead of creating; `Destroy` and `Status` treat nil as "nothing of a cloud's to act on". A registered machine is a full member of the sandbox registry either way — the registry lists machines spekk talks to, which is a different question from who made them.
