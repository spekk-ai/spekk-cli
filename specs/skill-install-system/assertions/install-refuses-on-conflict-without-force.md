---
id: install-refuses-on-conflict-without-force
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 1
status: done
depends-on: install-fetches-from-official-registry
branch: feature/skill-install-system
---

# Install Refuses to Overwrite Without --force

## Description

If the destination file already exists at the chosen scope, install fails with a clear error naming the path and instructing the user to pass `--force`. This protects local customizations from being silently clobbered and keeps the command deterministic for scripts/CI.

## Success Criteria

- When `<scope>/.spekk/skills/<agent>/<skill>.md` exists and `--force` is not set, the command exits non-zero
- The error message names the absolute conflicting path and mentions `--force` explicitly
- No HTTP request is made when the conflict is detected (existence is checked before fetching)
- The existing file is left unchanged on conflict
- With `--force`, the existing file is overwritten with the freshly fetched content
- `--force` works for both registry installs and `--source` installs

**Tests:** internal/install/write_test.go
