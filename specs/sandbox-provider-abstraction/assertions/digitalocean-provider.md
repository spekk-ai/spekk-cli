---
id: digitalocean-provider
parent: sandbox-provider-abstraction
created: 2026-08-29T20:00:00Z
priority: 1
status: done
depends-on: provider-interface
branch: dev/headless-sandbox
---

# DigitalOcean is a Provider implementation

The existing DO code (`doapi.go`) is wrapped in a struct that implements the `Provider` interface. No behavioral change to the DO sandbox flow — same API calls, same provisioning, same result.

## Success Criteria

- A `DOProvider` (or similar) struct implements the `Provider` interface
- `doapi.go` and its types remain largely intact — the provider wraps them, not rewrites them
- DO-specific config (region, size, VPC UUID, project) lives in the provider's Create config, not in generic `CreateOptions`
- `DOProvider.Create` orchestrates: SSH key upload, droplet creation, wait-for-active, return IP
- `DOProvider.Destroy` orchestrates: delete droplet, delete SSH key (handling 404s gracefully)
- `DOProvider.Status` fetches live droplet status via `GetDroplet`
- Provider stores its own state (droplet ID, SSH key ID) so it can destroy cleanly — this state is not in generic `SandboxMeta`
- `spekk sandbox create --provider digitalocean --name X --region nyc1 --size s-2vcpu-4gb` produces the same result as today's `spekk sandbox create --name X --region nyc1 --size s-2vcpu-4gb`
