---
id: spec-flag
parent: builder-cli-flags
created: 2026-02-21T12:00:00Z
priority: 1
status: done
---

# --spec Flag Filters to Specific Spec

## Description

The `--spec <id>` flag limits the builder to work only on assertions within the specified spec.

## Success Criteria

- `spekk builder --spec auth` only considers assertions under `specs/auth/`
- Respects priority ordering within that spec
- Combines with other flags:
  - `--spec auth --all` builds all assertions in auth spec
  - `--spec auth --dry-run` previews next assertion in auth spec
- Shows error if spec ID doesn't exist
