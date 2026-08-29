---
id: provider-interface
parent: sandbox-provider-abstraction
created: 2026-08-29T20:00:00Z
priority: 1
status: done
branch: feat/provider-interface
---

# A Provider Interface Abstracts Machine Lifecycle From Sandbox Orchestration

A sandbox has two halves. One half is the machine: create it, tear it down, ask how it is. That half is the provider's. The other half — waiting for provisioning, injecting credentials, deploying the agent, writing metadata — is the same whoever made the machine, and stays in `commands.go`. The interface is the line between them.

## Success Criteria

- A `Provider` interface in `internal/sandbox/` defines `Name`, `Create`, `Destroy`, and `Status`.
- Each method takes the sandbox's `*SandboxMeta`. `Create` fills the fields it owns — IP and SSH key path always, plus its own identifiers and any value it defaulted. `Destroy` and `Status` read them.
- `SandboxMeta` gains `Provider string` and keeps every field it already had, under the same JSON names. The change is additive: a metadata file written by an earlier binary loads with no loss.
- A provider's own identifiers live in named `SandboxMeta` fields, not in an opaque encoded handle. This is deliberate while one cloud provider exists: it keeps the file legible, it needs no encode step, and it keeps "this sandbox has no machine" distinguishable from "the identifier did not survive the load". A second cloud provider adds its own fields.
- `Create`, `Destroy`, and `Status` in `commands.go` take a `Provider` rather than constructing a DigitalOcean client inline.
- `Destroy` calls the provider unconditionally and returns its error. It never removes local metadata after a failed teardown, because that metadata is what makes an orphaned machine findable.
- `Destroy` removes a local key pair only when spekk generated it, judged by whether the path is inside the generated keys directory.
- `Status` accepts a nil provider and falls back to the stored state with a warning, so a missing API token degrades the command instead of failing it. A provider that cannot be built must come back as an untyped nil, or that fallback is skipped and the nil is called.
- `destroy` and `status` choose the provider by reading stored metadata. `deploy` and `ssh` need only an address and a key, so they stay provider-blind.
- `Create` records the machine as soon as the provider reports it exists, before the provisioning that can fail. A machine with no metadata entry is one `spekk sandbox destroy` cannot reach.

**Tests:** internal/sandbox/provider_test.go
