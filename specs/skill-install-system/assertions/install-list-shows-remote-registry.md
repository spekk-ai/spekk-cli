---
id: install-list-shows-remote-registry
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 2
status: in_progress
depends-on: install-fetches-from-official-registry
branch: feature/skill-install-system
locked-by: builder-Paris-MacBook-Pro.local-65115-1779479498
---

# `spekk install --list` Shows the Remote Registry

## Description

`spekk install --list <agent>` hits the GitHub contents API for the registry repo and prints every `.md` file in the `<agent>/` directory, marking which are already installed locally or globally. Useful for discovery before committing to an install.

## Success Criteria

- `spekk install --list coach` requests `<api-base>/coach` (where `<api-base>` honors `SPEKK_SKILLS_API_BASE`)
- The response is parsed as the GitHub contents JSON array; only entries with `type: "file"` and a `.md` suffix are listed
- Each skill is printed by its filename stem (no `.md` suffix)
- Each entry is annotated with whether it's already installed locally, globally, or not installed
- A network or parse error surfaces a clear message and exits non-zero
- 403 (rate-limit) is recognized and the error mentions the GitHub unauthenticated 60 req/hr limit
- The agent argument is required and validated the same way as install
- `spekk install --list` (no agent) exits non-zero with usage
