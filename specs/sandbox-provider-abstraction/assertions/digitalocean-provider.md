---
id: digitalocean-provider
parent: sandbox-provider-abstraction
created: 2026-08-29T20:00:00Z
priority: 1
status: done
depends-on: provider-interface
branch: feat/provider-interface
---

# DigitalOcean Is a Provider Implementation

The existing DigitalOcean code in `doapi.go` is wrapped in a struct that implements `Provider`. This is a refactor: same API calls, same order, same result, same output.

## Success Criteria

- A `DOProvider` struct implements `Provider`, and `doapi.go` and its types stay intact — the provider calls them rather than replacing them.
- `DOProvider.Create` resolves the project, generates and uploads an SSH key, renders cloud-init with the generated public key, creates the droplet, waits for its IP, and assigns the project.
- `DOProvider.Create` records what it resolved, not what the flags said: omitting `--region` and `--size` still leaves `nyc1` and `s-2vcpu-4gb` in metadata. A project given by UUID is still recorded as that UUID, because `resolveProject` returns its input in that case; naming it would need a second lookup that nothing yet asks for.
- `DOProvider.Create` records the droplet id and SSH key id as soon as the droplet exists, so a failure later in the flow still leaves a destroyable record.
- `DOProvider.Destroy` deletes the droplet and the SSH key, treating a 404 from either as already done rather than as an error.
- `DOProvider.Destroy` refuses, with an error naming the risk, when no droplet id is recorded. Deleting the metadata of a droplet that may still be running and billing is worse than stopping.
- `DOProvider.Status` fetches live droplet status through `GetDroplet`, and returns an empty string when there is no droplet id to ask about.
- The DigitalOcean flags — region, size, VPC, project — reach the provider through `CreateOptions`. A typed struct is used rather than a string map so the compiler still checks them and defaults cannot be lost in transit.

**Tests:** internal/sandbox/provider_test.go
