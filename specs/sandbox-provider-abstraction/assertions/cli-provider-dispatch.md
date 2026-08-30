---
id: cli-provider-dispatch
parent: sandbox-provider-abstraction
created: 2026-08-29T20:00:00Z
priority: 1
status: done
depends-on: digitalocean-provider
branch: feat/manual-sandbox
---

# Create Chooses Between Making a Machine and Using One That Exists

There are only two answers, and the flags already imply which one the operator wants. The job of this layer is to read that intent, refuse a request that asks for both at once, and keep the surface small enough that a third answer does not cost a redesign.

## Success Criteria

- `spekk sandbox create` accepts `--provider`, whose values are `digitalocean` and `none`. An unknown value is an error that lists the valid ones.
- With `--provider` omitted, naming an existing machine with `--ip` or `--ssh-key` resolves to `none`; anything else resolves to `digitalocean`.
- The cloud flags (`--region`, `--size`, `--vpc`, `--project`) and the existing-machine flags (`--ip`, `--ssh-key`) cannot be mixed. The error names the offending flags and the provider in force, including when that provider was inferred rather than typed.
- `ProviderByName` returns a nil `Provider` for `none`, and never a nil pointer inside a live interface for any value.
- `destroy` and `status` read the provider from stored metadata. A nil provider means no cloud owns the machine, so there is nothing of its to tear down and no live state to fetch; both commands fall back to what they can do over SSH.
