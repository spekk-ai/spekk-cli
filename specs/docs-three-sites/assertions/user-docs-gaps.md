---
id: user-docs-gaps
parent: docs-three-sites
created: 2026-06-04T14:00:00Z
priority: 2
status: not_started
branch: feature/docs-three-sites
---

# Existing user docs site covers troubleshooting, observer, and update command

The existing `docs/` Zensical site is enhanced with missing content identified by the observer.

## Success Criteria

- **Troubleshooting page** (`docs/troubleshooting.md`):
  - What happens when Claude CLI is not installed
  - `spekk next` returns empty but specs exist (branch filtering explanation)
  - `spekk serve` connection issues
  - `GITHUB_TOKEN` requirement for self-update
  - Added to `zensical.toml` navigation
- **Expanded observer section** in `docs/cli-reference.md`:
  - Comparable coverage to builder and coach sections
  - Documents drift detection, observation format, skill activation
  - How observer output feeds back into spec workflow
- **Update command section** in `docs/getting-started.md`:
  - "Updating Spekk" section documenting `spekk update` and `spekk update --check`
  - `GITHUB_TOKEN` requirement (fine-grained PAT with `contents:read`)
  - How to obtain and configure the token
