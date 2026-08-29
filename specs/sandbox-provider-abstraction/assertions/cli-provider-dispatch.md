---
id: cli-provider-dispatch
parent: sandbox-provider-abstraction
created: 2026-08-29T20:00:00Z
priority: 1
status: not_started
depends-on: digitalocean-provider
branch: dev/headless-sandbox
---

# CLI dispatches sandbox commands to the correct provider

## Success Criteria

- `spekk sandbox create` accepts `--provider <name>` flag (valid values: `digitalocean`, `manual`)
- When `--provider` is omitted and `--ip` is set, provider defaults to `manual`
- When `--provider` is omitted and `--ip` is not set, provider defaults to `digitalocean`
- Provider-specific flags are validated: `--region`, `--size`, `--vpc`, `--project` are errors with `--provider manual`; `--ip`, `--ssh-key` are errors with `--provider digitalocean`
- `spekk sandbox destroy`, `status`, `deploy`, and `ssh` read the provider from stored metadata and dispatch to the correct provider implementation — no `--provider` flag needed on these commands
- Error messages for invalid flag combinations name the conflicting flags and the selected provider
