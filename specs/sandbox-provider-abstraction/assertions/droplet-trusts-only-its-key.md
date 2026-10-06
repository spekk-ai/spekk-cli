---
id: droplet-trusts-only-its-key
parent: sandbox-provider-abstraction
created: 2026-10-06T00:00:00Z
priority: 1
status: done
depends-on: digitalocean-provider
branch: fix/droplet-trusts-only-its-key
---

# A Droplet Trusts Only the Key Spekk Generated for It

`DOProvider.Create` used to add every SSH key on the DigitalOcean account to the droplet, so that the operator could log in with their own key. Each of those keys logs in as root, and root reads `/etc/spekk/agent.env`, which holds the GitHub token and the model credential that spekk injects. In a team account, that gave every teammate the sandbox's credentials, and a key added to the account for an unrelated reason did the same.

The operator does not need their own key on the droplet. `spekk sandbox ssh <name>` logs in with the generated key, and the DigitalOcean console gives recovery access when that key is lost. The GCP provider makes the same choice when it blocks the project's SSH keys (see `gcp-provider`).

## Success Criteria

- `DOProvider.Create` creates the droplet with the generated key as its only SSH key, and does not list the keys on the account.
- `spekk sandbox ssh` still reaches the droplet, because it uses the generated key.

**Tests:** internal/sandbox/provider_test.go
