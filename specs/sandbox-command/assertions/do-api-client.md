---
id: do-api-client
parent: sandbox-command
created: 2026-03-12T15:00:00Z
priority: 1
status: not_started
depends-on: sandbox-command-routing
---

# DigitalOcean API Client

## Requirement

A thin API client in `src/sandbox/do-api.js` wraps DigitalOcean's REST API using Node.js native `fetch`. No external HTTP libraries or `doctl` dependency.

## Success Criteria

- `src/sandbox/do-api.js` exports functions for: `createDroplet`, `getDroplet`, `listDroplets`, `deleteDroplet`, `listSSHKeys`
- All functions read `DO_API_TOKEN` from `process.env` and throw a clear error if it is not set
- `createDroplet({ name, region, size, userData, sshKeyIds })` POSTs to `/v2/droplets` and returns the created droplet object
- `getDroplet(id)` GETs `/v2/droplets/{id}` and returns droplet status, IP addresses, and metadata
- `listDroplets(tag)` GETs `/v2/droplets?tag_name={tag}` to filter sandboxes by a `spekk-sandbox` tag
- `deleteDroplet(id)` DELETEs `/v2/droplets/{id}` and returns success/failure
- `listSSHKeys()` GETs `/v2/account/keys` and returns available SSH key IDs and fingerprints
- All API errors (4xx, 5xx) are caught and re-thrown with the DO error message included
- No dependencies added to `package.json` for HTTP -- uses global `fetch`
